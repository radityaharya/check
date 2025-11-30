package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GotifyNotifier struct {
	serverURL string
	token     string
	client    *http.Client
}

type GotifyMessage struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

func NewGotifyNotifier(serverURL, token string) *GotifyNotifier {
	serverURL = strings.TrimSuffix(serverURL, "/")
	return &GotifyNotifier{
		serverURL: serverURL,
		token:     token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (g *GotifyNotifier) GetServerURL() string {
	return g.serverURL
}

func (g *GotifyNotifier) TestWebhook() error {
	if g.serverURL == "" || g.token == "" {
		return fmt.Errorf("gotify server URL and token are required")
	}

	message := GotifyMessage{
		Title:    "GoCheck Test Notification",
		Message:  "If you see this message, your Gotify integration is configured correctly!",
		Priority: 4,
	}

	return g.sendMessage(message)
}

func (g *GotifyNotifier) SendStatusChange(checkName, url string, isUp bool, statusCode int, responseTimeMs int, errorMsg string) error {
	if g.serverURL == "" || g.token == "" {
		return nil
	}

	var status string
	var priority int
	if isUp {
		status = "UP"
		priority = 4
	} else {
		status = "DOWN"
		priority = 8
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("Status changed to **%s**\n\n", status))
	messageBuilder.WriteString(fmt.Sprintf("**URL:** %s\n", url))

	if statusCode > 0 {
		messageBuilder.WriteString(fmt.Sprintf("**Status Code:** %d\n", statusCode))
	}

	if responseTimeMs > 0 {
		messageBuilder.WriteString(fmt.Sprintf("**Response Time:** %d ms\n", responseTimeMs))
	}

	if errorMsg != "" {
		messageBuilder.WriteString(fmt.Sprintf("**Error:** %s\n", errorMsg))
	}

	message := GotifyMessage{
		Title:    fmt.Sprintf("Uptime Check: %s", checkName),
		Message:  messageBuilder.String(),
		Priority: priority,
	}

	return g.sendMessage(message)
}

func (g *GotifyNotifier) sendMessage(msg GotifyMessage) error {
	url := fmt.Sprintf("%s/message?token=%s", g.serverURL, g.token)

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify returned status %d", resp.StatusCode)
	}

	return nil
}

