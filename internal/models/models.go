package models

import (
	"encoding/json"
	"strconv"
	"time"
)

type CheckType string

type FlexibleInt64 struct {
	Value *int64
}

func (f *FlexibleInt64) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		f.Value = &i
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" || s == "null" {
			f.Value = nil
			return nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		f.Value = &i
		return nil
	}

	f.Value = nil
	return nil
}

func (f FlexibleInt64) MarshalJSON() ([]byte, error) {
	if f.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*f.Value)
}

// FlexibleInt handles int values that might come as strings from JSON
type FlexibleInt struct {
	Value int
	Set   bool
}

func (f *FlexibleInt) UnmarshalJSON(data []byte) error {
	f.Set = true
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		f.Value = i
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			f.Value = 0
			f.Set = false
			return nil
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		f.Value = i
		return nil
	}

	return nil
}

func (f FlexibleInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Value)
}

const (
	CheckTypeHTTP             CheckType = "http"
	CheckTypePing             CheckType = "ping"
	CheckTypePostgres         CheckType = "postgres"
	CheckTypeJSONHTTP         CheckType = "json_http"
	CheckTypeDNS              CheckType = "dns"
	CheckTypeTailscale        CheckType = "tailscale"
	CheckTypeTailscaleService CheckType = "tailscale_service"
)

type Group struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Check struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Type              CheckType `json:"type"`
	URL               string    `json:"url"`
	IntervalSeconds   int       `json:"interval_seconds"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Retries           int       `json:"retries,omitempty"`
	RetryDelaySeconds int       `json:"retry_delay_seconds,omitempty"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	GroupID           *int64    `json:"group_id,omitempty"`
	Tags              []Tag     `json:"tags,omitempty"`

	// HTTP specific
	ExpectedStatusCodes []int  `json:"expected_status_codes,omitempty"`
	Method              string `json:"method,omitempty"`

	// JSON HTTP specific - JSONata expression for assertion
	JSONPath          string `json:"json_path,omitempty"`
	ExpectedJSONValue string `json:"expected_json_value,omitempty"`

	// PostgreSQL specific
	PostgresConnString string `json:"postgres_conn_string,omitempty"`
	PostgresQuery      string `json:"postgres_query,omitempty"`
	ExpectedQueryValue string `json:"expected_query_value,omitempty"`

	// Ping specific
	Host string `json:"host,omitempty"`

	// DNS specific
	DNSHostname      string `json:"dns_hostname,omitempty"`
	DNSRecordType    string `json:"dns_record_type,omitempty"`
	ExpectedDNSValue string `json:"expected_dns_value,omitempty"`

	// Tailscale specific
	TailscaleDeviceID string `json:"tailscale_device_id,omitempty"`

	// Tailscale Service specific
	TailscaleServiceHost     string `json:"tailscale_service_host,omitempty"`
	TailscaleServicePort     int    `json:"tailscale_service_port,omitempty"`
	TailscaleServiceProtocol string `json:"tailscale_service_protocol,omitempty"` // http, https, tcp
	TailscaleServicePath     string `json:"tailscale_service_path,omitempty"`     // for http/https

	// Snapshot metadata
	SnapshotURL     string     `json:"snapshot_url,omitempty"`
	SnapshotTakenAt *time.Time `json:"snapshot_taken_at,omitempty"`
	SnapshotError   string     `json:"snapshot_error,omitempty"`
}

type CheckHistory struct {
	ID             int64     `json:"id"`
	CheckID        int64     `json:"check_id"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
	ResponseBody   string    `json:"response_body,omitempty"`
}

type CheckWithStatus struct {
	Check
	LastStatus    *CheckHistory  `json:"last_status,omitempty"`
	History       []CheckHistory `json:"history,omitempty"`
	IsUp          bool           `json:"is_up"`
	LastCheckedAt *time.Time     `json:"last_checked_at,omitempty"`
}

type GroupWithChecks struct {
	Group
	Checks    []CheckWithStatus `json:"checks"`
	IsUp      bool              `json:"is_up"`
	UpCount   int               `json:"up_count"`
	DownCount int               `json:"down_count"`
}

type Stats struct {
	TotalChecks  int     `json:"total_checks"`
	ActiveChecks int     `json:"active_checks"`
	UpChecks     int     `json:"up_checks"`
	DownChecks   int     `json:"down_checks"`
	TotalUptime  float64 `json:"total_uptime"`
}

type CreateCheckRequest struct {
	Name                string        `json:"name"`
	Type                CheckType     `json:"type"`
	URL                 string        `json:"url,omitempty"`
	IntervalSeconds     FlexibleInt   `json:"interval_seconds"`
	TimeoutSeconds      FlexibleInt   `json:"timeout_seconds"`
	Retries             FlexibleInt   `json:"retries"`
	RetryDelaySeconds   FlexibleInt   `json:"retry_delay_seconds"`
	Enabled             bool          `json:"enabled"`
	GroupID             FlexibleInt64 `json:"group_id,omitempty"`
	TagIDs              []int64       `json:"tag_ids,omitempty"`
	ExpectedStatusCodes []int         `json:"expected_status_codes,omitempty"`
	Method              string        `json:"method,omitempty"`
	JSONPath            string        `json:"json_path,omitempty"`
	ExpectedJSONValue   string        `json:"expected_json_value,omitempty"`
	PostgresConnString  string        `json:"postgres_conn_string,omitempty"`
	PostgresQuery       string        `json:"postgres_query,omitempty"`
	ExpectedQueryValue  string        `json:"expected_query_value,omitempty"`
	Host                string        `json:"host,omitempty"`
	DNSHostname         string        `json:"dns_hostname,omitempty"`
	DNSRecordType       string        `json:"dns_record_type,omitempty"`
	ExpectedDNSValue    string        `json:"expected_dns_value,omitempty"`
	TailscaleDeviceID   string        `json:"tailscale_device_id,omitempty"`
	TailscaleServiceHost     string   `json:"tailscale_service_host,omitempty"`
	TailscaleServicePort     FlexibleInt `json:"tailscale_service_port,omitempty"`
	TailscaleServiceProtocol string   `json:"tailscale_service_protocol,omitempty"`
	TailscaleServicePath     string   `json:"tailscale_service_path,omitempty"`
}

type UpdateCheckRequest struct {
	Name                *string       `json:"name,omitempty"`
	Type                *CheckType    `json:"type,omitempty"`
	URL                 *string       `json:"url,omitempty"`
	IntervalSeconds     FlexibleInt   `json:"interval_seconds,omitempty"`
	TimeoutSeconds      FlexibleInt   `json:"timeout_seconds,omitempty"`
	Retries             FlexibleInt   `json:"retries,omitempty"`
	RetryDelaySeconds   FlexibleInt   `json:"retry_delay_seconds,omitempty"`
	Enabled             *bool         `json:"enabled,omitempty"`
	GroupID             FlexibleInt64 `json:"group_id,omitempty"`
	TagIDs              *[]int64      `json:"tag_ids,omitempty"`
	ExpectedStatusCodes *[]int        `json:"expected_status_codes,omitempty"`
	Method              *string       `json:"method,omitempty"`
	JSONPath            *string       `json:"json_path,omitempty"`
	ExpectedJSONValue   *string       `json:"expected_json_value,omitempty"`
	PostgresConnString  *string       `json:"postgres_conn_string,omitempty"`
	PostgresQuery       *string       `json:"postgres_query,omitempty"`
	ExpectedQueryValue  *string       `json:"expected_query_value,omitempty"`
	Host                *string       `json:"host,omitempty"`
	DNSHostname         *string       `json:"dns_hostname,omitempty"`
	DNSRecordType       *string       `json:"dns_record_type,omitempty"`
	ExpectedDNSValue    *string       `json:"expected_dns_value,omitempty"`
	TailscaleDeviceID   *string       `json:"tailscale_device_id,omitempty"`
	TailscaleServiceHost     *string  `json:"tailscale_service_host,omitempty"`
	TailscaleServicePort     FlexibleInt `json:"tailscale_service_port,omitempty"`
	TailscaleServiceProtocol *string  `json:"tailscale_service_protocol,omitempty"`
	TailscaleServicePath     *string  `json:"tailscale_service_path,omitempty"`
}

type CreateGroupRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type UpdateGroupRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

type Settings struct {
	DiscordWebhookURL string `json:"discord_webhook_url"`
	GotifyServerURL   string `json:"gotify_server_url"`
	GotifyToken       string `json:"gotify_token"`
	TailscaleAPIKey   string `json:"tailscale_api_key"`
	TailscaleTailnet  string `json:"tailscale_tailnet"`
	CloudflareAccountID string `json:"cloudflare_account_id"`
	CloudflareAPIToken  string `json:"cloudflare_api_token"`
}

type CheckSnapshot struct {
	CheckID    int64      `json:"check_id"`
	FilePath   string     `json:"file_path,omitempty"`
	TakenAt    *time.Time `json:"taken_at,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
}

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type APIKey struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Key         string    `json:"key,omitempty"`
	KeyHash     string    `json:"-"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type Session struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type WebAuthnCredential struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CredentialID    []byte    `json:"credential_id"`
	PublicKey       []byte    `json:"public_key"`
	AttestationType string    `json:"attestation_type"`
	AAGUID          []byte    `json:"aaguid"`
	SignCount       uint32    `json:"sign_count"`
	CloneWarning    bool      `json:"clone_warning"`
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"created_at"`
}
