package checker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"gocheck/internal/models"

	tailscale "tailscale.com/client/tailscale/v2"
	"tailscale.com/tsnet"
)

var (
	tsnetServer   *tsnet.Server
	tsnetOnce     sync.Once
	tsnetInitErr  error
)

// getTsnetServer returns a singleton tsnet server instance
func getTsnetServer() (*tsnet.Server, error) {
	tsnetOnce.Do(func() {
		tsnetServer = &tsnet.Server{
			Hostname:  "gocheck-monitor",
			Dir:       "./data/tailscale",
			Ephemeral: true,
			Logf:      func(format string, args ...any) {}, // Silent logging
		}
	})
	return tsnetServer, tsnetInitErr
}

// performTailscaleCheck checks if a Tailscale device is online
func (e *Engine) performTailscaleCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.TailscaleDeviceID == "" {
		history.Success = false
		history.ErrorMessage = "no device ID specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	apiKey, _ := e.db.GetSetting("tailscale_api_key")
	tailnet, _ := e.db.GetSetting("tailscale_tailnet")

	if apiKey == "" || tailnet == "" {
		history.Success = false
		history.ErrorMessage = "Tailscale API key or tailnet not configured"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	client := &tailscale.Client{
		Tailnet: tailnet,
		APIKey:  apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	device, err := client.Devices().Get(ctx, check.TailscaleDeviceID)
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("API error: %v", err)
		return
	}

	// Check if device is online via connectedToControl
	if device.ConnectedToControl {
		history.Success = true
		history.ResponseBody = fmt.Sprintf("Online - %s (%s)", device.Hostname, strings.Join(device.Addresses, ", "))
		return
	}

	// Check lastSeen - if within 5 minutes, consider online
	lastSeenTime := device.LastSeen.Time
	if !lastSeenTime.IsZero() {
		lastSeenDuration := time.Since(lastSeenTime)
		if lastSeenDuration < 5*time.Minute {
			history.Success = true
			history.ResponseBody = fmt.Sprintf("Online (last seen %s ago) - %s", lastSeenDuration.Round(time.Second), device.Hostname)
			return
		}
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("Device offline - last seen %s ago", lastSeenDuration.Round(time.Second))
		return
	}

	history.Success = false
	history.ErrorMessage = "Device status unknown"
}

// performTailscaleServiceCheck checks if a service running on a Tailscale device is accessible
func (e *Engine) performTailscaleServiceCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.TailscaleServiceHost == "" {
		history.Success = false
		history.ErrorMessage = "no Tailscale host specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	if check.TailscaleServicePort == 0 {
		history.Success = false
		history.ErrorMessage = "no port specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	// Get tsnet server
	srv, err := getTsnetServer()
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("failed to initialize Tailscale: %v", err)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	// Determine the protocol
	protocol := check.TailscaleServiceProtocol
	if protocol == "" {
		protocol = "http"
	}

	// For HTTP/HTTPS checks
	if protocol == "http" || protocol == "https" {
		path := check.TailscaleServicePath
		if path == "" {
			path = "/"
		}
		url := fmt.Sprintf("%s://%s:%d%s", protocol, check.TailscaleServiceHost, check.TailscaleServicePort, path)

		// Create HTTP client that uses tsnet
		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return srv.Dial(ctx, network, addr)
				},
			},
			Timeout: time.Duration(check.TimeoutSeconds) * time.Second,
		}

		method := check.Method
		if method == "" {
			method = "GET"
		}

		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("invalid request: %v", err)
			history.ResponseTimeMs = int(time.Since(start).Milliseconds())
			return
		}

		resp, err := httpClient.Do(req)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())

		if err != nil {
			history.Success = false
			history.ErrorMessage = err.Error()
			history.StatusCode = 0
			return
		}
		defer resp.Body.Close()

		history.StatusCode = resp.StatusCode

		expectedStatusCodes := check.ExpectedStatusCodes
		if len(expectedStatusCodes) == 0 {
			expectedStatusCodes = []int{200}
		}

		success := false
		for _, expectedCode := range expectedStatusCodes {
			if resp.StatusCode == expectedCode {
				success = true
				break
			}
		}

		if !success && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			success = true
		}

		if success {
			history.Success = true
			history.ResponseBody = fmt.Sprintf("%s service responding on %s:%d", protocol, check.TailscaleServiceHost, check.TailscaleServicePort)
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("unexpected status code: %d (expected: %v)", resp.StatusCode, expectedStatusCodes)
		}
		return
	}

	// For TCP checks
	if protocol == "tcp" {
		target := fmt.Sprintf("%s:%d", check.TailscaleServiceHost, check.TailscaleServicePort)
		conn, err := srv.Dial(ctx, "tcp", target)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())

		if err != nil {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("TCP connection failed: %v", err)
			return
		}
		defer conn.Close()

		history.Success = true
		history.ResponseBody = fmt.Sprintf("TCP connection successful to %s:%d", check.TailscaleServiceHost, check.TailscaleServicePort)
		return
	}

	history.Success = false
	history.ErrorMessage = fmt.Sprintf("unsupported protocol: %s", protocol)
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())
}
