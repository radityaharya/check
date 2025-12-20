package snapshot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"

	"gocheck/internal/checker"
	"gocheck/internal/db"
	"gocheck/internal/models"
)

const refreshInterval = 6 * time.Hour

type Service struct {
	db            *db.Database
	engine        *checker.Engine
	dataDir       string
	screenshotDir string
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	sem chan struct{}
}

func NewService(database *db.Database, engine *checker.Engine, dataDir string) *Service {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		absDataDir = dataDir
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		db:            database,
		engine:        engine,
		dataDir:       absDataDir,
		screenshotDir: filepath.Join(absDataDir, "screenshots"),
		ctx:           ctx,
		cancel:        cancel,
		sem:           make(chan struct{}, 1),
	}
}

func (s *Service) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *Service) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *Service) TriggerRefresh() {
	go s.refreshAll()
}

func (s *Service) CaptureCheck(checkID int64) error {
	check, err := s.db.GetCheck(checkID)
	if err != nil || check == nil {
		return fmt.Errorf("check %d not found in database", checkID)
	}

	if s.isTailscale(*check) {
		log.Printf("snapshot: skipping check %d (Tailscale/Private network logic ignored)", checkID)
		return nil
	}

	targetURL, err := s.resolveTargetURL(*check)
	if err != nil {
		s.storeFailure(checkID, "", err.Error())
		return err
	}

	data, err := s.performCapture(targetURL)
	if err != nil {
		s.storeFailure(checkID, "", err.Error())
		return err
	}

	if err := os.MkdirAll(s.screenshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	filePath := filepath.Join(s.screenshotDir, fmt.Sprintf("check_%d.png", checkID))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	now := time.Now().UTC()
	err = s.db.UpsertCheckSnapshot(&models.CheckSnapshot{
		CheckID:   checkID,
		FilePath:  filePath,
		TakenAt:   &now,
		LastError: "",
	})

	s.broadcastSnapshot(checkID)
	return err
}

func (s *Service) TestSnapshot(targetURL string) ([]byte, error) {
	return s.performCapture(targetURL)
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
	checks, err := s.db.GetAllChecks()
	if err != nil {
		log.Printf("snapshot: failed to list checks for refresh: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, check := range checks {
		if s.isTailscale(check) {
			continue
		}

		snapshot, err := s.db.GetCheckSnapshot(check.ID)
		if err != nil {
			log.Printf("snapshot: failed to get snapshot for check %d: %v", check.ID, err)
			continue
		}

		needsRefresh := false
		if snapshot == nil || snapshot.TakenAt == nil {
			needsRefresh = true
		} else {
			age := now.Sub(*snapshot.TakenAt)
			if age >= refreshInterval {
				needsRefresh = true
			}
		}

		if needsRefresh {
			_ = s.CaptureCheck(check.ID)
		}
	}
}

func (s *Service) performCapture(targetURL string) (data []byte, err error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}

	bURL, token, err := s.loadCredentials()
	if err != nil || bURL == "" {
		return nil, fmt.Errorf("browserless credentials missing from settings")
	}

	controlURL, err := s.buildBrowserlessURL(bURL, token)
	if err != nil {
		return nil, fmt.Errorf("failed to build browserless URL: %w", err)
	}

	// Use a shorter timeout than browserless (which is set to 120s)
	// This ensures we close the browser BEFORE browserless times out and sends 429
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	data, err = s.executeCapture(ctx, controlURL, targetURL)
	return data, err
}

func (s *Service) executeCapture(ctx context.Context, controlURL, targetURL string) ([]byte, error) {
	// Create browser with context for automatic cancellation
	browser := rod.New().ControlURL(controlURL).Context(ctx)
	
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("browserless connection failed (check URL and token): %w", err)
	}
	
	// IMPORTANT: Close browser synchronously to terminate WebSocket before browserless can send 429
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to open browser page: %w", err)
	}
	defer page.Close()

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1280,
		Height: 800,
	}); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	if err := page.Navigate(targetURL); err != nil {
		return nil, fmt.Errorf("navigation to %s failed: %w", targetURL, err)
	}

	// Wait for page to fully load
	if err := page.WaitLoad(); err != nil {
		// Log but don't fail - page might still be usable
		log.Printf("snapshot: WaitLoad warning for %s: %v", targetURL, err)
	}

	// Wait for network to be idle (no requests for 500ms)
	if err := page.WaitIdle(10 * time.Second); err != nil {
		log.Printf("snapshot: WaitIdle warning for %s: %v", targetURL, err)
	}

	// Wait for DOM content to be loaded and rendered
	// This helps with SPAs and JS-heavy sites
	_ = page.WaitDOMStable(2*time.Second, 0.1)

	// Additional wait for any animations, lazy-loaded images, etc.
	time.Sleep(3 * time.Second)

	quality := 90
	screenshot, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: &quality,
	})
	if err != nil {
		return nil, fmt.Errorf("screenshot capture failed: %w", err)
	}

	return screenshot, nil
}

func (s *Service) buildBrowserlessURL(rawURL, token string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	token = strings.TrimSpace(token)

	if rawURL == "" || token == "" {
		return "", fmt.Errorf("browserless credentials cannot be empty")
	}

	isSecure := strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "wss://")
	
	cleanHost := rawURL
	prefixes := []string{"https://", "http://", "wss://", "ws://"}
	for _, p := range prefixes {
		cleanHost = strings.TrimPrefix(cleanHost, p)
	}
	cleanHost = strings.TrimRight(cleanHost, "/")

	scheme := "ws://"
	if isSecure {
		scheme = "wss://"
	}

	u, err := url.Parse(fmt.Sprintf("%s%s/chromium", scheme, cleanHost))
	if err != nil {
		return "", fmt.Errorf("invalid browserless url: %w", err)
	}

	// Build launch args as JSON for browserless v2
	// Use a LONGER timeout for browserless than our Go context, so we can close cleanly
	launchArgs := `{"args":["--disable-dev-shm-usage","--no-sandbox"],"timeout":120000}`

	q := u.Query()
	q.Set("token", token)
	q.Set("launch", launchArgs)
	q.Set("timeout", "120000") // 120 seconds - longer than Go timeout so we close first
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (s *Service) resolveTargetURL(check models.Check) (string, error) {
	var targetURL string

	if check.URL != "" {
		targetURL = check.URL
	} else if check.Type == models.CheckTypeTailscaleService && check.TailscaleServiceHost != "" {
		port := check.TailscaleServicePort
		if port == 0 {
			port = 80
		}
		path := "/" + strings.TrimPrefix(check.TailscaleServicePath, "/")
		targetURL = fmt.Sprintf("%s://%s:%d%s", check.TailscaleServiceProtocol, check.TailscaleServiceHost, port, path)
	}

	if targetURL == "" {
		return "", fmt.Errorf("no valid URL or Tailscale host defined for check")
	}

	// Validate the URL
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", targetURL, err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid URL %q: missing host", targetURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid URL %q: scheme must be http or https", targetURL)
	}

	return targetURL, nil
}

func (s *Service) isTailscale(check models.Check) bool {
	if check.Type == models.CheckTypeTailscaleService {
		return true
	}
	
	lowURL := strings.ToLower(check.URL)
	lowHost := strings.ToLower(check.TailscaleServiceHost)
	
	return strings.Contains(lowURL, ".ts.net") || 
		   strings.Contains(lowHost, ".ts.net") ||
		   strings.HasPrefix(lowHost, "100.")
}


func (s *Service) loadCredentials() (string, string, error) {
	u, _ := s.db.GetSetting("browserless_url")
	t, _ := s.db.GetSetting("browserless_token")
	return u, t, nil
}

func (s *Service) storeFailure(checkID int64, filePath, message string) {
	log.Printf("snapshot: check %d failed: %s", checkID, message)
	_ = s.db.UpsertCheckSnapshot(&models.CheckSnapshot{
		CheckID:   checkID,
		FilePath:  filePath,
		LastError: message,
	})
	s.broadcastSnapshot(checkID)
}

func (s *Service) broadcastSnapshot(checkID int64) {
	if s.engine == nil {
		return
	}
	check, err := s.db.GetCheck(checkID)
	if err == nil && check != nil {
		s.engine.BroadcastCheckSnapshot(*check)
	}
}