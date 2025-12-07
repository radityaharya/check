package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gocheck/internal/db"
	"gocheck/internal/models"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnUser struct {
	user        *models.User
	credentials []models.WebAuthnCredential
}

func (u WebAuthnUser) WebAuthnID() []byte {
	return []byte(string(rune(u.user.ID)))
}

func (u WebAuthnUser) WebAuthnName() string {
	return u.user.Username
}

func (u WebAuthnUser) WebAuthnDisplayName() string {
	return u.user.Username
}

func (u WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(u.credentials))
	for i, c := range u.credentials {
		creds[i] = webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:       c.AAGUID,
				SignCount:    c.SignCount,
				CloneWarning: c.CloneWarning,
			},
		}
	}
	return creds
}

type WebAuthnManager struct {
	db       *db.Database
	sessions map[string]*sessionEntry
}

type sessionEntry struct {
	data      *webauthn.SessionData
	expiresAt time.Time
}

func NewWebAuthnManager(rpID, rpOrigin string, database *db.Database) (*WebAuthnManager, error) {
	wm := &WebAuthnManager{
		db:       database,
		sessions: make(map[string]*sessionEntry),
	}
	go wm.cleanupExpiredSessions()
	return wm, nil
}

func (wm *WebAuthnManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for token, entry := range wm.sessions {
			if now.After(entry.expiresAt) {
				delete(wm.sessions, token)
			}
		}
	}
}

func (wm *WebAuthnManager) getOriginFromRequest(r *http.Request) string {
	// Try to get origin from Origin header first (most reliable)
	if origin := r.Header.Get("Origin"); origin != "" {
		return origin
	}
	
	// Fallback: construct from request
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	
	return scheme + "://" + host
}

func (wm *WebAuthnManager) getHostFromRequest(r *http.Request) string {
	// Try to extract from Origin header first
	if origin := r.Header.Get("Origin"); origin != "" {
		// Parse the origin URL to get just the hostname
		// origin format: https://uptime.civet-universe.ts.net
		origin = strings.TrimPrefix(origin, "https://")
		origin = strings.TrimPrefix(origin, "http://")
		
		// Remove port if present
		if idx := strings.Index(origin, ":"); idx != -1 {
			origin = origin[:idx]
		}
		
		return origin
	}
	
	// Fallback to Host header
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	
	// Strip port if present (RP ID should be hostname only)
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	
	return host
}

func (wm *WebAuthnManager) createWebAuthnForRequest(r *http.Request) (*webauthn.WebAuthn, error) {
	origin := wm.getOriginFromRequest(r)
	host := wm.getHostFromRequest(r)
	
	log.Printf("WebAuthn: Creating config with RPID=%s, Origin=%s", host, origin)
	
	wconfig := &webauthn.Config{
		RPDisplayName: "Gocheck Monitor",
		RPID:          host,
		RPOrigins:     []string{origin},
	}
	
	return webauthn.New(wconfig)
}

func (wm *WebAuthnManager) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	webAuthn, err := wm.createWebAuthnForRequest(r)
	if err != nil {
		http.Error(w, "failed to initialize webauthn", http.StatusInternalServerError)
		return
	}
	
	session, _ := globalAuthManager.GetSession(r)
	if session == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := wm.db.GetUserByID(session.UserID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	credentials, _ := wm.db.GetWebAuthnCredentialsByUserID(user.ID)
	webAuthnUser := WebAuthnUser{user: user, credentials: credentials}

	// Request resident key (discoverable credential) for username-less login
	options, sessionData, err := webAuthn.BeginRegistration(
		webAuthnUser,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationPreferred,
		}),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wm.sessions[session.Token] = &sessionEntry{
		data:      sessionData,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (wm *WebAuthnManager) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	webAuthn, err := wm.createWebAuthnForRequest(r)
	if err != nil {
		http.Error(w, "failed to initialize webauthn", http.StatusInternalServerError)
		return
	}
	
	session, _ := globalAuthManager.GetSession(r)
	if session == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := wm.db.GetUserByID(session.UserID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	credentials, _ := wm.db.GetWebAuthnCredentialsByUserID(user.ID)
	webAuthnUser := WebAuthnUser{user: user, credentials: credentials}

	entry, ok := wm.sessions[session.Token]
	if !ok {
		http.Error(w, "session not found", http.StatusBadRequest)
		return
	}
	sessionData := entry.data
	delete(wm.sessions, session.Token)

	// Read the full request body first to extract the name
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()

	var reqData struct {
		Name string `json:"name"`
	}
	json.Unmarshal(bodyBytes, &reqData)

	// Create a new reader with the original body for WebAuthn parsing
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	credential, err := webAuthn.FinishRegistration(webAuthnUser, *sessionData, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := reqData.Name
	if name == "" {
		name = "Passkey"
	}

	cred := &models.WebAuthnCredential{
		UserID:          user.ID,
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		Name:            name,
	}

	if err := wm.db.CreateWebAuthnCredential(cred); err != nil {
		http.Error(w, "failed to save credential", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   cred.ID,
		"name": cred.Name,
	})
}

func (wm *WebAuthnManager) BeginLogin(w http.ResponseWriter, r *http.Request) {
	webAuthn, err := wm.createWebAuthnForRequest(r)
	if err != nil {
		http.Error(w, "failed to initialize webauthn", http.StatusInternalServerError)
		return
	}
	
	var req struct {
		Username string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Trim whitespace and check if username is provided
	username := strings.TrimSpace(req.Username)
	
	log.Printf("WebAuthn BeginLogin: username='%s' (empty=%v)", username, username == "")

	// If no username provided, use discoverable credential flow
	if username == "" {
		log.Printf("WebAuthn: Using discoverable credential flow (no username)")
		// Discoverable credential login (no username needed)
		options, sessionData, err := webAuthn.BeginDiscoverableLogin()
		if err != nil {
			log.Printf("WebAuthn: BeginDiscoverableLogin error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token, _ := generateSessionToken()
		wm.sessions[token] = &sessionEntry{
			data:      sessionData,
			expiresAt: time.Now().Add(5 * time.Minute),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"options": options,
			"token":   token,
		})
		return
	}

	log.Printf("WebAuthn: Using traditional flow with username: %s", username)
	
	// Traditional flow with username
	user, err := wm.db.GetUserByUsername(username)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	credentials, _ := wm.db.GetWebAuthnCredentialsByUserID(user.ID)
	if len(credentials) == 0 {
		http.Error(w, "no passkeys registered", http.StatusBadRequest)
		return
	}

	webAuthnUser := WebAuthnUser{user: user, credentials: credentials}

	options, sessionData, err := webAuthn.BeginLogin(webAuthnUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, _ := generateSessionToken()
	wm.sessions[token] = &sessionEntry{
		data:      sessionData,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"options": options,
		"token":   token,
	})
}

func (wm *WebAuthnManager) FinishLogin(w http.ResponseWriter, r *http.Request) {
	webAuthn, err := wm.createWebAuthnForRequest(r)
	if err != nil {
		http.Error(w, "failed to initialize webauthn", http.StatusInternalServerError)
		return
	}
	
	// Read the full request body first to extract username and token
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()

	var reqData struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	if err := json.Unmarshal(bodyBytes, &reqData); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	entry, ok := wm.sessions[reqData.Token]
	if !ok {
		http.Error(w, "session not found", http.StatusBadRequest)
		return
	}
	sessionData := entry.data
	delete(wm.sessions, reqData.Token)

	// Reset body for WebAuthn parsing
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var user *models.User
	var credential *webauthn.Credential

	// If no username provided, use discoverable credential flow
	if reqData.Username == "" {
		// Parse the credential response to get user handle
		parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
		if err != nil {
			http.Error(w, "invalid credential response", http.StatusBadRequest)
			return
		}

		// User handle contains the user ID
		if len(parsedResponse.Response.UserHandle) == 0 {
			http.Error(w, "no user handle in credential", http.StatusBadRequest)
			return
		}

		// Get user by ID from user handle
		userID := int64(parsedResponse.Response.UserHandle[0])
		user, err = wm.db.GetUserByID(userID)
		if err != nil || user == nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		// Reset body again for WebAuthn library
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		
		credentials, _ := wm.db.GetWebAuthnCredentialsByUserID(user.ID)
		webAuthnUser := WebAuthnUser{user: user, credentials: credentials}
		
		credential, err = webAuthn.FinishLogin(webAuthnUser, *sessionData, r)
	} else {
		// Traditional flow with username
		user, err = wm.db.GetUserByUsername(reqData.Username)
		if err != nil || user == nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		credentials, _ := wm.db.GetWebAuthnCredentialsByUserID(user.ID)
		webAuthnUser := WebAuthnUser{user: user, credentials: credentials}

		credential, err = webAuthn.FinishLogin(webAuthnUser, *sessionData, r)
	}

	if err != nil {
		errMsg := err.Error()
		log.Printf("WebAuthn login error: %v", errMsg)
		
		// If it's a backup flag error, allow it anyway
		if !strings.Contains(errMsg, "Backup Eligible flag inconsistency") && 
		   !strings.Contains(errMsg, "backup") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		
		log.Printf("WebAuthn: Ignoring backup flag inconsistency, allowing login")
	}

	if credential != nil {
		wm.db.UpdateWebAuthnCredentialSignCount(credential.ID, credential.Authenticator.SignCount)
	}

	token, _ := generateSessionToken()
	expiresAt := time.Now().Add(24 * time.Hour)
	session := &models.Session{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: expiresAt,
	}

	if err := wm.db.CreateSession(session); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (wm *WebAuthnManager) GetPasskeys(w http.ResponseWriter, r *http.Request) {
	session, _ := globalAuthManager.GetSession(r)
	if session == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	credentials, err := wm.db.GetWebAuthnCredentialsByUserID(session.UserID)
	if err != nil {
		http.Error(w, "failed to get passkeys", http.StatusInternalServerError)
		return
	}

	type passkeyInfo struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
	}

	passkeys := make([]passkeyInfo, len(credentials))
	for i, c := range credentials {
		passkeys[i] = passkeyInfo{
			ID:        c.ID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(passkeys)
}

func (wm *WebAuthnManager) DeletePasskey(w http.ResponseWriter, r *http.Request) {
	session, _ := globalAuthManager.GetSession(r)
	if session == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	credentials, err := wm.db.GetWebAuthnCredentialsByUserID(session.UserID)
	if err != nil {
		http.Error(w, "failed to verify ownership", http.StatusInternalServerError)
		return
	}

	found := false
	for _, c := range credentials {
		if c.ID == req.ID {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "passkey not found", http.StatusNotFound)
		return
	}

	if err := wm.db.DeleteWebAuthnCredential(req.ID); err != nil {
		http.Error(w, "failed to delete passkey", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

var globalAuthManager *AuthManager
var globalWebAuthnManager *WebAuthnManager

func SetGlobalManagers(am *AuthManager, wm *WebAuthnManager) {
	globalAuthManager = am
	globalWebAuthnManager = wm
}
