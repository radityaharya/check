package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gocheck/internal/checker"
	"gocheck/internal/db"
	"gocheck/internal/models"
)

const refreshInterval = 6 * time.Hour

// Service periodically captures webpage screenshots for checks using Cloudflare Browser Rendering.
type Service struct {
	db            *db.Database
	engine        *checker.Engine
	client        *http.Client
	dataDir       string
	screenshotDir string
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewService builds a snapshot service with a default HTTP client and paths.
func NewService(database *db.Database, engine *checker.Engine, dataDir string) *Service {
	absDataDir, err := filepath.Abs(dataDir)
	if err == nil {
		dataDir = absDataDir
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:            database,
		engine:        engine,
		client:        &http.Client{Timeout: 60 * time.Second},
		dataDir:       dataDir,
		screenshotDir: filepath.Join(dataDir, "screenshots"),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the periodic refresh loop with an initial capture attempt.
func (s *Service) Start() {
	s.wg.Add(1)
	go s.run()
}

// Stop halts the refresh loop.
func (s *Service) Stop() {
	s.cancel()
	s.wg.Wait()
}

// TriggerRefresh launches a refresh cycle asynchronously.
func (s *Service) TriggerRefresh() {
	go s.refreshAll()
}

// CaptureCheck runs a one-off snapshot for a single check ID.
func (s *Service) CaptureCheck(checkID int64) error {
	accountID, apiToken, err := s.loadCredentials()
	if err != nil {
		log.Printf("snapshot: load credentials error: %v", err)
		return err
	}
	if accountID == "" || apiToken == "" {
		return fmt.Errorf("cloudflare credentials not configured")
	}

	if err := os.MkdirAll(s.screenshotDir, 0755); err != nil {
		return fmt.Errorf("prepare screenshot dir: %w", err)
	}

	check, err := s.db.GetCheck(checkID)
	if err != nil {
		return err
	}
	if check == nil {
		return fmt.Errorf("check not found")
	}

	targetURL, err := s.resolveTargetURL(*check)
	if err != nil {
		s.storeFailure(checkID, "", err.Error())
		return err
	}

	if err := s.captureSnapshot(checkID, targetURL, accountID, apiToken); err != nil {
		return err
	}

	s.broadcastSnapshot(checkID)
	return nil
}

func (s *Service) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	s.refreshAll()

	for {
		select {
		case <-ticker.C:
			s.refreshAll()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) refreshAll() {
	accountID, apiToken, err := s.loadCredentials()
	if err != nil {
		return
	}
	if accountID == "" || apiToken == "" {
		return
	}

	if err := os.MkdirAll(s.screenshotDir, 0755); err != nil {
		return
	}

	checks, err := s.db.GetAllChecks()
	if err != nil {
		return
	}

	for _, check := range checks {
		targetURL, err := s.resolveTargetURL(check)
		if err != nil {
			s.storeFailure(check.ID, "", err.Error())
			continue
		}
		_ = s.captureSnapshot(check.ID, targetURL, accountID, apiToken)
		s.broadcastSnapshot(check.ID)
	}
}

func (s *Service) loadCredentials() (string, string, error) {
	accountID, err := s.db.GetSetting("cloudflare_account_id")
	if err != nil {
		return "", "", err
	}
	apiToken, err := s.db.GetSetting("cloudflare_api_token")
	if err != nil {
		return "", "", err
	}
	return accountID, apiToken, nil
}

func (s *Service) resolveTargetURL(check models.Check) (string, error) {
	if check.URL != "" {
		parsed, err := url.Parse(check.URL)
		if err != nil {
			return "", fmt.Errorf("invalid url: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", fmt.Errorf("snapshot supports http/https only")
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("url missing host")
		}
		return parsed.String(), nil
	}

	if check.Type == models.CheckTypeTailscaleService && check.TailscaleServiceHost != "" {
		if check.TailscaleServiceProtocol == "http" || check.TailscaleServiceProtocol == "https" {
			portPart := ""
			if check.TailscaleServicePort > 0 {
				portPart = fmt.Sprintf(":%d", check.TailscaleServicePort)
			}
			path := check.TailscaleServicePath
			if path == "" {
				path = "/"
			}
			built := fmt.Sprintf("%s://%s%s%s", check.TailscaleServiceProtocol, check.TailscaleServiceHost, portPart, path)
			parsed, err := url.Parse(built)
			if err != nil {
				return "", fmt.Errorf("invalid tailscale service url: %w", err)
			}
			if parsed.Host == "" {
				return "", fmt.Errorf("tailscale service url missing host")
			}
			return parsed.String(), nil
		}
	}

	return "", fmt.Errorf("snapshot available only for http/https checks with a valid URL")
}

func (s *Service) captureSnapshot(checkID int64, targetURL, accountID, apiToken string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"url": targetURL,
		"gotoOptions": map[string]interface{}{
			"waitUntil": "networkidle0",
			"timeout":   45000,
		},
		"screenshotOptions": map[string]bool{
			"omitBackground": true,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		s.storeFailure(checkID, "", fmt.Sprintf("encode payload: %v", err))
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/screenshot", accountID),
		&buf,
	)
	if err != nil {
		s.storeFailure(checkID, "", fmt.Sprintf("build request: %v", err))
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.storeFailure(checkID, "", fmt.Sprintf("request error: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		errMsg := fmt.Sprintf("cloudflare status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		s.storeFailure(checkID, "", errMsg)
		return errors.New(errMsg)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		errMsg := fmt.Sprintf("unexpected content type %s: %s", contentType, html.EscapeString(string(body)))
		s.storeFailure(checkID, "", errMsg)
		return errors.New(errMsg)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		s.storeFailure(checkID, "", fmt.Sprintf("read response: %v", err))
		return err
	}

	filePath := filepath.Join(s.screenshotDir, fmt.Sprintf("check_%d.png", checkID))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		s.storeFailure(checkID, filePath, fmt.Sprintf("write file: %v", err))
		return err
	}

	now := time.Now().UTC()
	if err := s.db.UpsertCheckSnapshot(&models.CheckSnapshot{
		CheckID:   checkID,
		FilePath:  filePath,
		TakenAt:   &now,
		LastError: "",
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) storeFailure(checkID int64, filePath, message string) {
	log.Printf("snapshot: check %d failed: %s", checkID, message)
	existing, _ := s.db.GetCheckSnapshot(checkID)
	snapshot := &models.CheckSnapshot{
		CheckID:   checkID,
		FilePath:  filePath,
		LastError: message,
	}
	if existing != nil {
		if snapshot.FilePath == "" {
			snapshot.FilePath = existing.FilePath
		}
		if existing.TakenAt != nil {
			snapshot.TakenAt = existing.TakenAt
		}
	}
	_ = s.db.UpsertCheckSnapshot(snapshot)
	s.broadcastSnapshot(checkID)
}

func (s *Service) broadcastSnapshot(checkID int64) {
	if s.engine == nil {
		return
	}
	check, err := s.db.GetCheck(checkID)
	if err != nil || check == nil {
		return
	}
	s.engine.BroadcastCheckSnapshot(*check)
}
