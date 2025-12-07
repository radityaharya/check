package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"gocheck/internal/checker"
	"gocheck/internal/db"
	"gocheck/internal/models"
	"gocheck/internal/notifier"
	tailscale "tailscale.com/client/tailscale/v2"
)

type Handlers struct {
	db        *db.Database
	engine    *checker.Engine
	notifiers []notifier.Notifier
}

func NewHandlers(database *db.Database, engine *checker.Engine, notifiers []notifier.Notifier) *Handlers {
	return &Handlers{
		db:        database,
		engine:    engine,
		notifiers: notifiers,
	}
}

func parseRangeParam(r *http.Request) (*time.Time, error) {
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		return nil, nil
	}

	var dur time.Duration
	switch rangeStr {
	case "15m":
		dur = 15 * time.Minute
	case "30m":
		dur = 30 * time.Minute
	case "60m":
		dur = 60 * time.Minute
	case "1d":
		dur = 24 * time.Hour
	case "30d":
		dur = 30 * 24 * time.Hour
	default:
		return nil, fmt.Errorf("invalid range")
	}

	t := time.Now().Add(-dur)
	return &t, nil
}

func (h *Handlers) GetChecks(w http.ResponseWriter, r *http.Request) {
	since, err := parseRangeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Determine aggregation strategy based on time range
	var historyLimit int
	var bucketMinutes int

	if since == nil {
		// No range specified, use recent raw data
		historyLimit = 50
	} else {
		// Calculate time range duration
		duration := time.Since(*since)

		if duration <= 1*time.Hour {
			// For ranges <= 1 hour, use raw data
			historyLimit = 500
			bucketMinutes = 0
		} else if duration <= 24*time.Hour {
			// For ranges <= 1 day, aggregate by 5-minute buckets
			historyLimit = 288
			bucketMinutes = 5
		} else if duration <= 7*24*time.Hour {
			// For ranges <= 7 days, aggregate by 1-hour buckets
			historyLimit = 168
			bucketMinutes = 60
		} else {
			// For ranges > 7 days, aggregate by 6-hour buckets
			historyLimit = 120
			bucketMinutes = 360
		}
	}

	checks, err := h.db.GetAllChecks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var checksWithStatus []models.CheckWithStatus
	for _, check := range checks {
		var history []models.CheckHistory
		lastStatus, _ := h.db.GetLastStatus(check.ID)

		// Use aggregated or raw history based on time range
		if bucketMinutes > 0 {
			history, _ = h.db.GetCheckHistoryAggregated(check.ID, since, bucketMinutes, historyLimit)
		} else {
			history, _ = h.db.GetCheckHistory(check.ID, since, historyLimit)
		}

		cws := models.CheckWithStatus{
			Check:      check,
			LastStatus: lastStatus,
			History:    history,
		}

		if lastStatus != nil {
			cws.IsUp = lastStatus.Success
			cws.LastCheckedAt = &lastStatus.CheckedAt
		}

		checksWithStatus = append(checksWithStatus, cws)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checksWithStatus)
}

func (h *Handlers) CreateCheck(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = models.CheckTypeHTTP
	}

	intervalSeconds := req.IntervalSeconds.Value
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	timeoutSeconds := req.TimeoutSeconds.Value
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	retries := req.Retries.Value
	if retries < 0 {
		retries = 0
	}
	if retries > 10 {
		retries = 10
	}
	retryDelaySeconds := req.RetryDelaySeconds.Value
	if retryDelaySeconds <= 0 {
		retryDelaySeconds = 5
	}
	if retryDelaySeconds > 60 {
		retryDelaySeconds = 60
	}

	check := models.Check{
		Name:                     req.Name,
		Type:                     req.Type,
		URL:                      req.URL,
		IntervalSeconds:          intervalSeconds,
		TimeoutSeconds:           timeoutSeconds,
		Retries:                  retries,
		RetryDelaySeconds:        retryDelaySeconds,
		Enabled:                  req.Enabled,
		GroupID:                  req.GroupID.Value,
		ExpectedStatusCodes:      req.ExpectedStatusCodes,
		Method:                   req.Method,
		JSONPath:                 req.JSONPath,
		ExpectedJSONValue:        req.ExpectedJSONValue,
		PostgresConnString:       req.PostgresConnString,
		PostgresQuery:            req.PostgresQuery,
		ExpectedQueryValue:       req.ExpectedQueryValue,
		Host:                     req.Host,
		DNSHostname:              req.DNSHostname,
		DNSRecordType:            req.DNSRecordType,
		ExpectedDNSValue:         req.ExpectedDNSValue,
		TailscaleDeviceID:        req.TailscaleDeviceID,
		TailscaleServiceHost:     req.TailscaleServiceHost,
		TailscaleServicePort:     req.TailscaleServicePort.Value,
		TailscaleServiceProtocol: req.TailscaleServiceProtocol,
		TailscaleServicePath:     req.TailscaleServicePath,
	}

	if check.Method == "" {
		check.Method = "GET"
	}
	if len(check.ExpectedStatusCodes) == 0 {
		check.ExpectedStatusCodes = []int{200}
	}
	if check.DNSRecordType == "" && check.Type == models.CheckTypeDNS {
		check.DNSRecordType = "A"
	}

	if err := h.db.CreateCheck(&check); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(req.TagIDs) > 0 {
		h.db.SetCheckTags(check.ID, req.TagIDs)
		check.Tags, _ = h.db.GetCheckTags(check.ID)
	}

	h.engine.AddCheck(check)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(check)
}

func (h *Handlers) UpdateCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	check, err := h.db.GetCheck(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if check == nil {
		http.Error(w, "check not found", http.StatusNotFound)
		return
	}

	var req models.UpdateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		check.Name = *req.Name
	}
	if req.Type != nil {
		check.Type = *req.Type
	}
	if req.URL != nil {
		check.URL = *req.URL
	}
	if req.IntervalSeconds.Set {
		check.IntervalSeconds = req.IntervalSeconds.Value
	}
	if req.TimeoutSeconds.Set {
		check.TimeoutSeconds = req.TimeoutSeconds.Value
	}
	if req.Retries.Set {
		value := req.Retries.Value
		if value < 0 {
			value = 0
		}
		if value > 10 {
			value = 10
		}
		check.Retries = value
	}
	if req.RetryDelaySeconds.Set {
		value := req.RetryDelaySeconds.Value
		if value <= 0 {
			value = 5
		}
		if value > 60 {
			value = 60
		}
		check.RetryDelaySeconds = value
	}
	if req.Enabled != nil {
		check.Enabled = *req.Enabled
	}
	check.GroupID = req.GroupID.Value
	if req.ExpectedStatusCodes != nil {
		check.ExpectedStatusCodes = *req.ExpectedStatusCodes
	}
	if req.Method != nil {
		check.Method = *req.Method
	}
	if req.JSONPath != nil {
		check.JSONPath = *req.JSONPath
	}
	if req.ExpectedJSONValue != nil {
		check.ExpectedJSONValue = *req.ExpectedJSONValue
	}
	if req.PostgresConnString != nil {
		check.PostgresConnString = *req.PostgresConnString
	}
	if req.PostgresQuery != nil {
		check.PostgresQuery = *req.PostgresQuery
	}
	if req.ExpectedQueryValue != nil {
		check.ExpectedQueryValue = *req.ExpectedQueryValue
	}
	if req.Host != nil {
		check.Host = *req.Host
	}
	if req.DNSHostname != nil {
		check.DNSHostname = *req.DNSHostname
	}
	if req.DNSRecordType != nil {
		check.DNSRecordType = *req.DNSRecordType
	}
	if req.ExpectedDNSValue != nil {
		check.ExpectedDNSValue = *req.ExpectedDNSValue
	}
	if req.TailscaleDeviceID != nil {
		check.TailscaleDeviceID = *req.TailscaleDeviceID
	}
	if req.TailscaleServiceHost != nil {
		check.TailscaleServiceHost = *req.TailscaleServiceHost
	}
	if req.TailscaleServicePort.Set {
		check.TailscaleServicePort = req.TailscaleServicePort.Value
	}
	if req.TailscaleServiceProtocol != nil {
		check.TailscaleServiceProtocol = *req.TailscaleServiceProtocol
	}
	if req.TailscaleServicePath != nil {
		check.TailscaleServicePath = *req.TailscaleServicePath
	}

	if err := h.db.UpdateCheck(check); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.TagIDs != nil {
		h.db.SetCheckTags(check.ID, *req.TagIDs)
		check.Tags, _ = h.db.GetCheckTags(check.ID)
	}

	h.engine.AddCheck(*check)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}

func (h *Handlers) DeleteCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteCheck(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.RemoveCheck(id)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetCheckHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	since, err := parseRangeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var history []models.CheckHistory

	// Determine aggregation strategy based on time range
	if since != nil {
		duration := time.Since(*since)

		if duration <= 1*time.Hour {
			// For ranges <= 1 hour, use raw data
			history, err = h.db.GetCheckHistory(id, since, limit)
		} else if duration <= 24*time.Hour {
			// For ranges <= 1 day, aggregate by 5-minute buckets
			history, err = h.db.GetCheckHistoryAggregated(id, since, 5, 288)
		} else if duration <= 7*24*time.Hour {
			// For ranges <= 7 days, aggregate by 1-hour buckets
			history, err = h.db.GetCheckHistoryAggregated(id, since, 60, 168)
		} else {
			// For ranges > 7 days, aggregate by 6-hour buckets
			history, err = h.db.GetCheckHistoryAggregated(id, since, 360, 120)
		}
	} else {
		history, err = h.db.GetCheckHistory(id, since, limit)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	since, err := parseRangeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats, err := h.db.GetStats(since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	webhookURL, _ := h.db.GetSetting("discord_webhook_url")
	gotifyServerURL, _ := h.db.GetSetting("gotify_server_url")
	gotifyToken, _ := h.db.GetSetting("gotify_token")
	tailscaleAPIKey, _ := h.db.GetSetting("tailscale_api_key")
	tailscaleTailnet, _ := h.db.GetSetting("tailscale_tailnet")

	settings := models.Settings{
		DiscordWebhookURL: webhookURL,
		GotifyServerURL:   gotifyServerURL,
		GotifyToken:       gotifyToken,
		TailscaleAPIKey:   tailscaleAPIKey,
		TailscaleTailnet:  tailscaleTailnet,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings models.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.SetSetting("discord_webhook_url", settings.DiscordWebhookURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("gotify_server_url", settings.GotifyServerURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("gotify_token", settings.GotifyToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("tailscale_api_key", settings.TailscaleAPIKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("tailscale_tailnet", settings.TailscaleTailnet); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var notifiers []notifier.Notifier
	if settings.DiscordWebhookURL != "" {
		notifiers = append(notifiers, notifier.NewDiscordNotifier(settings.DiscordWebhookURL))
	}
	if settings.GotifyServerURL != "" && settings.GotifyToken != "" {
		notifiers = append(notifiers, notifier.NewGotifyNotifier(settings.GotifyServerURL, settings.GotifyToken))
	}
	h.notifiers = notifiers
	h.engine.UpdateNotifiers(notifiers)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *Handlers) TestWebhook(w http.ResponseWriter, r *http.Request) {
	var discordNotifier *notifier.DiscordNotifier
	for _, n := range h.notifiers {
		if dn, ok := n.(*notifier.DiscordNotifier); ok {
			discordNotifier = dn
			break
		}
	}

	if discordNotifier == nil {
		http.Error(w, "discord notifier not configured", http.StatusBadRequest)
		return
	}

	if err := discordNotifier.TestWebhook(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Test notification sent successfully"})
}

func (h *Handlers) TestGotify(w http.ResponseWriter, r *http.Request) {
	var gotifyNotifier *notifier.GotifyNotifier
	for _, n := range h.notifiers {
		if gn, ok := n.(*notifier.GotifyNotifier); ok {
			gotifyNotifier = gn
			break
		}
	}

	if gotifyNotifier == nil {
		http.Error(w, "gotify notifier not configured", http.StatusBadRequest)
		return
	}

	if err := gotifyNotifier.TestWebhook(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Test notification sent successfully"})
}

func (h *Handlers) GetTailscaleDevices(w http.ResponseWriter, r *http.Request) {
	apiKey, _ := h.db.GetSetting("tailscale_api_key")
	tailnet, _ := h.db.GetSetting("tailscale_tailnet")

	if apiKey == "" || tailnet == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tailscale API key or tailnet not configured"})
		return
	}

	client := &tailscale.Client{
		Tailnet: tailnet,
		APIKey:  apiKey,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	devices, err := client.Devices().List(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	type DeviceInfo struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Hostname  string   `json:"hostname"`
		Addresses []string `json:"addresses"`
		Online    bool     `json:"online"`
		OS        string   `json:"os"`
		LastSeen  string   `json:"last_seen"`
	}

	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		online := d.ConnectedToControl
		if !online && !d.LastSeen.Time.IsZero() {
			online = time.Since(d.LastSeen.Time) < 5*time.Minute
		}
		lastSeen := ""
		if !d.LastSeen.Time.IsZero() {
			lastSeen = d.LastSeen.Time.Format(time.RFC3339)
		}
		result = append(result, DeviceInfo{
			ID:        d.ID,
			Name:      d.Name,
			Hostname:  d.Hostname,
			Addresses: d.Addresses,
			Online:    online,
			OS:        d.OS,
			LastSeen:  lastSeen,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) TestTailscale(w http.ResponseWriter, r *http.Request) {
	apiKey, _ := h.db.GetSetting("tailscale_api_key")
	tailnet, _ := h.db.GetSetting("tailscale_tailnet")

	if apiKey == "" || tailnet == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tailscale API key or tailnet not configured"})
		return
	}

	client := &tailscale.Client{
		Tailnet: tailnet,
		APIKey:  apiKey,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	devices, err := client.Devices().List(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "ok",
		"message":      "Connected to Tailscale successfully",
		"device_count": len(devices),
	})
}

func (h *Handlers) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.db.GetAllGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []models.Group{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func (h *Handlers) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group := models.Group{Name: req.Name, SortOrder: req.SortOrder}
	if err := h.db.CreateGroup(&group); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (h *Handlers) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	group, err := h.db.GetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if group == nil {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	var req models.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := h.db.UpdateGroup(group); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func (h *Handlers) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteGroup(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.db.GetAllTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tags == nil {
		tags = []models.Tag{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *Handlers) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Color == "" {
		req.Color = "#6b7280"
	}

	tag := models.Tag{Name: req.Name, Color: req.Color}
	if err := h.db.CreateTag(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (h *Handlers) UpdateTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	tag, err := h.db.GetTag(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag == nil {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}

	var req models.UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		tag.Name = *req.Name
	}
	if req.Color != nil {
		tag.Color = *req.Color
	}

	if err := h.db.UpdateTag(tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (h *Handlers) DeleteTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteTag(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetGroupedChecks(w http.ResponseWriter, r *http.Request) {
	since, err := parseRangeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Determine aggregation strategy based on time range
	var historyLimit int
	var bucketMinutes int

	if since == nil {
		// No range specified, use recent raw data
		historyLimit = 200
		bucketMinutes = 0
	} else {
		// Calculate time range duration
		duration := time.Since(*since)

		if duration <= 15*time.Minute {
			// For 15m range, fetch plenty of raw data
			historyLimit = 200
			bucketMinutes = 0
		} else if duration <= 30*time.Minute {
			// For 30m range, fetch plenty of raw data
			historyLimit = 300
			bucketMinutes = 0
		} else if duration <= 1*time.Hour {
			// For 1h range, fetch plenty of raw data
			historyLimit = 500
			bucketMinutes = 0
		} else if duration <= 24*time.Hour {
			// For 1d range, use all raw data (no aggregation)
			historyLimit = 2000
			bucketMinutes = 0
		} else if duration <= 7*24*time.Hour {
			// For 7d range, aggregate by 30-minute buckets
			historyLimit = 336 // 7 days * 48 (30-min buckets per day)
			bucketMinutes = 30
		} else {
			// For ranges > 7 days (30d), aggregate by 2-hour buckets
			historyLimit = 360 // 30 days * 12 (2-hour buckets per day)
			bucketMinutes = 120
		}
	}

	checks, err := h.db.GetAllChecks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groups, err := h.db.GetAllGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupMap := make(map[int64]*models.GroupWithChecks)
	for _, g := range groups {
		groupMap[g.ID] = &models.GroupWithChecks{
			Group:  g,
			Checks: []models.CheckWithStatus{},
			IsUp:   true,
		}
	}

	ungrouped := &models.GroupWithChecks{
		Group:  models.Group{ID: 0, Name: "Ungrouped"},
		Checks: []models.CheckWithStatus{},
		IsUp:   true,
	}

	// Fetch last statuses and histories concurrently to avoid N+1 latency
	lastStatusMap := make(map[int64]*models.CheckHistory)
	historyMap := make(map[int64][]models.CheckHistory)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// Limit concurrent DB calls to avoid exhausting connections
	maxWorkers := 8
	sem := make(chan struct{}, maxWorkers)

	for _, check := range checks {
		wg.Add(1)
		sem <- struct{}{}
		c := check
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			lastStatus, err := h.db.GetLastStatus(c.ID)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			var history []models.CheckHistory
			if bucketMinutes > 0 {
				history, err = h.db.GetCheckHistoryAggregated(c.ID, since, bucketMinutes, historyLimit)
			} else {
				history, err = h.db.GetCheckHistory(c.ID, since, historyLimit)
			}
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			lastStatusMap[c.ID] = lastStatus
			historyMap[c.ID] = history
			mu.Unlock()
		}()
	}

	wg.Wait()
	if firstErr != nil {
		http.Error(w, firstErr.Error(), http.StatusInternalServerError)
		return
	}

	for _, check := range checks {
		lastStatus := lastStatusMap[check.ID]
		history := historyMap[check.ID]

		cws := models.CheckWithStatus{
			Check:      check,
			LastStatus: lastStatus,
			History:    history,
		}

		if lastStatus != nil {
			cws.IsUp = lastStatus.Success
			cws.LastCheckedAt = &lastStatus.CheckedAt
		}

		if check.GroupID != nil {
			if g, ok := groupMap[*check.GroupID]; ok {
				g.Checks = append(g.Checks, cws)
				// Only count enabled checks towards group status
				if check.Enabled {
					if !cws.IsUp {
						g.IsUp = false
						g.DownCount++
					} else {
						g.UpCount++
					}
				}
			} else {
				ungrouped.Checks = append(ungrouped.Checks, cws)
				if check.Enabled {
					if !cws.IsUp {
						ungrouped.IsUp = false
						ungrouped.DownCount++
					} else {
						ungrouped.UpCount++
					}
				}
			}
		} else {
			ungrouped.Checks = append(ungrouped.Checks, cws)
			if check.Enabled {
				if !cws.IsUp {
					ungrouped.IsUp = false
					ungrouped.DownCount++
				} else {
					ungrouped.UpCount++
				}
			}
		}
	}

	result := []models.GroupWithChecks{}
	for _, g := range groups {
		if gwc, ok := groupMap[g.ID]; ok {
			result = append(result, *gwc)
		}
	}
	if len(ungrouped.Checks) > 0 {
		result = append(result, *ungrouped)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) StreamCheckUpdates(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to check updates
	client := h.engine.Subscribe()
	defer h.engine.Unsubscribe(client)

	// Get flusher for sending data
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"connected\"}\n\n")
	flusher.Flush()

	// Stream updates
	for {
		select {
		case event := <-client:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: check_update\ndata: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}
