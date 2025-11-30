package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

type DiscordEmbed struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Color       int          `json:"color"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordWebhook struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *DiscordNotifier) GetWebhookURL() string {
	return d.webhookURL
}

func (d *DiscordNotifier) TestWebhook() error {
	if d.webhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	embed := DiscordEmbed{
		Title:       "GoCheck Test Notification",
		Description: "If you see this message, your Discord webhook is configured correctly!",
		Color:       5814783,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []EmbedField{
			{Name: "Status", Value: "Test Successful", Inline: true},
		},
	}

	webhook := DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	}

	payload, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	req, err := http.NewRequest("POST", d.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (d *DiscordNotifier) SendStatusChange(checkName, url string, isUp bool, statusCode int, responseTimeMs int, errorMsg string) error {
	if d.webhookURL == "" {
		return nil
	}

	var color int
	var status string
	if isUp {
		color = 3066993
		status = "UP"
	} else {
		color = 15158332
		status = "DOWN"
	}

	embed := DiscordEmbed{
		Title:       fmt.Sprintf("Uptime Check: %s", checkName),
		Description: fmt.Sprintf("Status changed to **%s**", status),
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []EmbedField{
			{Name: "URL", Value: url, Inline: false},
			{Name: "Status", Value: status, Inline: true},
		},
	}

	if statusCode > 0 {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Status Code",
			Value:  fmt.Sprintf("%d", statusCode),
			Inline: true,
		})
	}

	if responseTimeMs > 0 {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Response Time",
			Value:  fmt.Sprintf("%d ms", responseTimeMs),
			Inline: true,
		})
	}

	if errorMsg != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Error",
			Value:  errorMsg,
			Inline: false,
		})
	}

	webhook := DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	}

	payload, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	req, err := http.NewRequest("POST", d.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

