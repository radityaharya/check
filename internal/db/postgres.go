package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gocheck/internal/models"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(connString string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool for PostgreSQL
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	d := &PostgresDB{db: db}
	if err := d.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

func (d *PostgresDB) Close() error {
	return d.db.Close()
}

func (d *PostgresDB) initSchema() error {
	// Create tables with proper PostgreSQL types and constraints
	schema := `
	-- Groups table
	CREATE TABLE IF NOT EXISTS groups (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Tags table
	CREATE TABLE IF NOT EXISTS tags (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#6b7280'
	);

	-- Checks table with comprehensive indexing
	CREATE TABLE IF NOT EXISTS checks (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'http',
		url TEXT,
		interval_seconds INTEGER NOT NULL DEFAULT 60,
		timeout_seconds INTEGER NOT NULL DEFAULT 10,
		retries INTEGER NOT NULL DEFAULT 0,
		retry_delay_seconds INTEGER NOT NULL DEFAULT 5,
		enabled BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expected_status_codes JSONB DEFAULT '[200]',
		method TEXT DEFAULT 'GET',
		json_path TEXT,
		expected_json_value TEXT,
		postgres_conn_string TEXT,
		postgres_query TEXT,
		expected_query_value TEXT,
		host TEXT,
		dns_hostname TEXT,
		dns_record_type TEXT,
		expected_dns_value TEXT,
		tailscale_device_id TEXT,
		tailscale_service_host TEXT,
		tailscale_service_port INTEGER,
		tailscale_service_protocol TEXT,
		tailscale_service_path TEXT,
		group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL
	);

	-- Check history table with partitioning support
	CREATE TABLE IF NOT EXISTS check_history (
		id BIGSERIAL PRIMARY KEY,
		check_id BIGINT NOT NULL,
		status_code INTEGER,
		response_time_ms INTEGER,
		success BOOLEAN NOT NULL,
		error_message TEXT,
		response_body TEXT,
		checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (check_id) REFERENCES checks(id) ON DELETE CASCADE
	);

	-- Check tags junction table
	CREATE TABLE IF NOT EXISTS check_tags (
		check_id BIGINT NOT NULL,
		tag_id BIGINT NOT NULL,
		PRIMARY KEY (check_id, tag_id),
		FOREIGN KEY (check_id) REFERENCES checks(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- API Keys table
	CREATE TABLE IF NOT EXISTS api_keys (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		last_used_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		id BIGSERIAL PRIMARY KEY,
		token TEXT NOT NULL UNIQUE,
		user_id BIGINT NOT NULL,
		username TEXT NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	-- WebAuthn Credentials table
	CREATE TABLE IF NOT EXISTS webauthn_credentials (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		credential_id BYTEA NOT NULL UNIQUE,
		public_key BYTEA NOT NULL,
		attestation_type TEXT NOT NULL,
		aaguid BYTEA,
		sign_count INTEGER NOT NULL DEFAULT 0,
		clone_warning BOOLEAN NOT NULL DEFAULT false,
		name TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_webauthn_creds_user_id ON webauthn_credentials(user_id);
	CREATE INDEX IF NOT EXISTS idx_webauthn_creds_credential_id ON webauthn_credentials(credential_id);

	-- Indexes for checks table
	CREATE INDEX IF NOT EXISTS idx_checks_enabled ON checks(enabled) WHERE enabled = true;
	CREATE INDEX IF NOT EXISTS idx_checks_created_at ON checks(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_checks_group_id ON checks(group_id) WHERE group_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_checks_type ON checks(type);

	-- Indexes for check_history table (optimized for time-series queries)
	CREATE INDEX IF NOT EXISTS idx_check_history_check_id ON check_history(check_id);
	CREATE INDEX IF NOT EXISTS idx_check_history_checked_at ON check_history(checked_at DESC);
	CREATE INDEX IF NOT EXISTS idx_check_history_check_id_checked_at ON check_history(check_id, checked_at DESC);
	CREATE INDEX IF NOT EXISTS idx_check_history_success ON check_history(success);
	
	-- Composite index for common query patterns
	CREATE INDEX IF NOT EXISTS idx_check_history_check_success_time ON check_history(check_id, success, checked_at DESC);

	-- Index for tags
	CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

	-- Add new columns if they don't exist (for migrations)
	DO $$ 
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='checks' AND column_name='tailscale_service_host') THEN
			ALTER TABLE checks ADD COLUMN tailscale_service_host TEXT;
		END IF;
		
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='checks' AND column_name='tailscale_service_port') THEN
			ALTER TABLE checks ADD COLUMN tailscale_service_port INTEGER;
		END IF;
		
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='checks' AND column_name='tailscale_service_protocol') THEN
			ALTER TABLE checks ADD COLUMN tailscale_service_protocol TEXT;
		END IF;
		
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='checks' AND column_name='tailscale_service_path') THEN
			ALTER TABLE checks ADD COLUMN tailscale_service_path TEXT;
		END IF;
	END $$;
	`

	_, err := d.db.Exec(schema)
	return err
}

func (d *PostgresDB) parseStatusCodes(data interface{}) []int {
	if data == nil {
		return []int{200}
	}

	var codes []int
	switch v := data.(type) {
	case []byte:
		if err := json.Unmarshal(v, &codes); err != nil {
			return []int{200}
		}
	case string:
		if err := json.Unmarshal([]byte(v), &codes); err != nil {
			return []int{200}
		}
	default:
		return []int{200}
	}

	if len(codes) == 0 {
		return []int{200}
	}
	return codes
}

func (d *PostgresDB) encodeStatusCodes(codes []int) []byte {
	if len(codes) == 0 {
		return []byte("[200]")
	}
	data, _ := json.Marshal(codes)
	return data
}

func (d *PostgresDB) GetAllChecks() ([]models.Check, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, COALESCE(url, ''), interval_seconds, timeout_seconds, retries, retry_delay_seconds, 
			enabled, created_at, COALESCE(expected_status_codes::text, '[200]'), method, 
			COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), 
			COALESCE(host, ''), COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), 
			COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, ''), 
			COALESCE(tailscale_service_host, ''), COALESCE(tailscale_service_port, 0), 
			COALESCE(tailscale_service_protocol, ''), COALESCE(tailscale_service_path, '')
		FROM checks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.Check
	for rows.Next() {
		var c models.Check
		var statusCodesJSON string
		var groupID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, 
			&c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
			&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
			&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
			&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID, &c.TailscaleServiceHost, &c.TailscaleServicePort, &c.TailscaleServiceProtocol, &c.TailscaleServicePath); err != nil {
			return nil, err
		}
		c.ExpectedStatusCodes = d.parseStatusCodes(statusCodesJSON)
		if groupID.Valid {
			c.GroupID = &groupID.Int64
		}
		c.Tags, _ = d.GetCheckTags(c.ID)
		checks = append(checks, c)
	}

	return checks, rows.Err()
}

func (d *PostgresDB) GetCheck(id int64) (*models.Check, error) {
	var c models.Check
	var statusCodesJSON string
	var groupID sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, name, type, COALESCE(url, ''), interval_seconds, timeout_seconds, retries, retry_delay_seconds, 
			enabled, created_at, COALESCE(expected_status_codes::text, '[200]'), method, 
			COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), 
			COALESCE(host, ''), COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), 
			COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, ''), 
			COALESCE(tailscale_service_host, ''), COALESCE(tailscale_service_port, 0), 
			COALESCE(tailscale_service_protocol, ''), COALESCE(tailscale_service_path, '')
		FROM checks
		WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, 
		&c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
		&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
		&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
		&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID,
		&c.TailscaleServiceHost, &c.TailscaleServicePort, &c.TailscaleServiceProtocol, &c.TailscaleServicePath)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.ExpectedStatusCodes = d.parseStatusCodes(statusCodesJSON)
	if groupID.Valid {
		c.GroupID = &groupID.Int64
	}
	c.Tags, _ = d.GetCheckTags(c.ID)
	return &c, nil
}

func (d *PostgresDB) CreateCheck(c *models.Check) error {
	statusCodesJSON := d.encodeStatusCodes(c.ExpectedStatusCodes)
	err := d.db.QueryRow(`
		INSERT INTO checks (name, type, url, interval_seconds, timeout_seconds, retries, retry_delay_seconds, 
			enabled, expected_status_codes, method, json_path, expected_json_value,
			postgres_conn_string, postgres_query, expected_query_value, host,
			dns_hostname, dns_record_type, expected_dns_value, group_id, tailscale_device_id,
			tailscale_service_host, tailscale_service_port, tailscale_service_protocol, tailscale_service_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id, created_at
	`, c.Name, c.Type, c.URL, c.IntervalSeconds, c.TimeoutSeconds, c.Retries, c.RetryDelaySeconds, 
		c.Enabled, statusCodesJSON, c.Method, c.JSONPath, c.ExpectedJSONValue,
		c.PostgresConnString, c.PostgresQuery, c.ExpectedQueryValue, c.Host,
		c.DNSHostname, c.DNSRecordType, c.ExpectedDNSValue, c.GroupID, c.TailscaleDeviceID,
		c.TailscaleServiceHost, c.TailscaleServicePort, c.TailscaleServiceProtocol, c.TailscaleServicePath).Scan(&c.ID, &c.CreatedAt)

	return err
}

func (d *PostgresDB) UpdateCheck(c *models.Check) error {
	statusCodesJSON := d.encodeStatusCodes(c.ExpectedStatusCodes)
	_, err := d.db.Exec(`
		UPDATE checks
		SET name = $1, type = $2, url = $3, interval_seconds = $4, timeout_seconds = $5, 
			retries = $6, retry_delay_seconds = $7, enabled = $8, expected_status_codes = $9, 
			method = $10, json_path = $11, expected_json_value = $12,
			postgres_conn_string = $13, postgres_query = $14, expected_query_value = $15, host = $16,
			dns_hostname = $17, dns_record_type = $18, expected_dns_value = $19, group_id = $20, 
			tailscale_device_id = $21, tailscale_service_host = $22, tailscale_service_port = $23,
			tailscale_service_protocol = $24, tailscale_service_path = $25
		WHERE id = $26
	`, c.Name, c.Type, c.URL, c.IntervalSeconds, c.TimeoutSeconds, c.Retries, c.RetryDelaySeconds, 
		c.Enabled, statusCodesJSON, c.Method, c.JSONPath, c.ExpectedJSONValue,
		c.PostgresConnString, c.PostgresQuery, c.ExpectedQueryValue, c.Host,
		c.DNSHostname, c.DNSRecordType, c.ExpectedDNSValue, c.GroupID, c.TailscaleDeviceID,
		c.TailscaleServiceHost, c.TailscaleServicePort, c.TailscaleServiceProtocol, c.TailscaleServicePath, c.ID)
	return err
}

func (d *PostgresDB) DeleteCheck(id int64) error {
	_, err := d.db.Exec("DELETE FROM checks WHERE id = $1", id)
	return err
}

func (d *PostgresDB) GetEnabledChecks() ([]models.Check, error) {
	rows, err := d.db.Query(`
		SELECT id, name, type, COALESCE(url, ''), interval_seconds, timeout_seconds, retries, retry_delay_seconds, 
			enabled, created_at, COALESCE(expected_status_codes::text, '[200]'), method, 
			COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), 
			COALESCE(host, ''), COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), 
			COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, ''), 
			COALESCE(tailscale_service_host, ''), COALESCE(tailscale_service_port, 0), 
			COALESCE(tailscale_service_protocol, ''), COALESCE(tailscale_service_path, '')
		FROM checks
		WHERE enabled = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.Check
	for rows.Next() {
		var c models.Check
		var statusCodesJSON string
		var groupID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, 
			&c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
			&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
			&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
			&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID, &c.TailscaleServiceHost, &c.TailscaleServicePort, &c.TailscaleServiceProtocol, &c.TailscaleServicePath); err != nil {
			return nil, err
		}
		c.ExpectedStatusCodes = d.parseStatusCodes(statusCodesJSON)
		if groupID.Valid {
			c.GroupID = &groupID.Int64
		}
		checks = append(checks, c)
	}

	return checks, rows.Err()
}

func (d *PostgresDB) AddHistory(h *models.CheckHistory) error {
	_, err := d.db.Exec(`
		INSERT INTO check_history (check_id, status_code, response_time_ms, success, error_message, response_body)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, h.CheckID, h.StatusCode, h.ResponseTimeMs, h.Success, h.ErrorMessage, h.ResponseBody)
	return err
}

func (d *PostgresDB) GetCheckHistory(checkID int64, since *time.Time, limit int) ([]models.CheckHistory, error) {
	query := `
		SELECT id, check_id, status_code, response_time_ms, success, COALESCE(error_message, ''), checked_at
		FROM check_history
		WHERE check_id = $1`
	args := []interface{}{checkID}

	if since != nil {
		query += " AND checked_at >= $2"
		args = append(args, since.UTC())
	}

	query += " ORDER BY checked_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.CheckHistory
	for rows.Next() {
		var h models.CheckHistory
		if err := rows.Scan(&h.ID, &h.CheckID, &h.StatusCode, &h.ResponseTimeMs, &h.Success, &h.ErrorMessage, &h.CheckedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

func (d *PostgresDB) GetCheckHistoryAggregated(checkID int64, since *time.Time, bucketMinutes int, limit int) ([]models.CheckHistory, error) {
	// Use PostgreSQL's date_trunc for better performance
	query := `
		SELECT 
			MAX(id) as id,
			check_id,
			CAST(AVG(status_code) AS INTEGER) as status_code,
			CAST(AVG(response_time_ms) AS INTEGER) as response_time_ms,
			BOOL_AND(success) as success,
			'' as error_message,
			date_trunc('minute', checked_at - INTERVAL '0 minute' * (EXTRACT(MINUTE FROM checked_at)::INTEGER % $1)) as checked_at
		FROM check_history
		WHERE check_id = $2`
	args := []interface{}{bucketMinutes, checkID}

	if since != nil {
		query += " AND checked_at >= $3"
		args = append(args, since.UTC())
	}

	query += " GROUP BY check_id, date_trunc('minute', checked_at - INTERVAL '0 minute' * (EXTRACT(MINUTE FROM checked_at)::INTEGER % $1)) ORDER BY checked_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.CheckHistory
	for rows.Next() {
		var h models.CheckHistory
		if err := rows.Scan(&h.ID, &h.CheckID, &h.StatusCode, &h.ResponseTimeMs, &h.Success, &h.ErrorMessage, &h.CheckedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

func (d *PostgresDB) GetLastStatus(checkID int64) (*models.CheckHistory, error) {
	var h models.CheckHistory
	err := d.db.QueryRow(`
		SELECT id, check_id, status_code, response_time_ms, success, COALESCE(error_message, ''), checked_at
		FROM check_history
		WHERE check_id = $1
		ORDER BY checked_at DESC
		LIMIT 1
	`, checkID).Scan(&h.ID, &h.CheckID, &h.StatusCode, &h.ResponseTimeMs, &h.Success, &h.ErrorMessage, &h.CheckedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &h, nil
}

func (d *PostgresDB) GetStats(since *time.Time) (*models.Stats, error) {
	var stats models.Stats

	err := d.db.QueryRow("SELECT COUNT(*) FROM checks").Scan(&stats.TotalChecks)
	if err != nil {
		return nil, err
	}

	err = d.db.QueryRow("SELECT COUNT(*) FROM checks WHERE enabled = true").Scan(&stats.ActiveChecks)
	if err != nil {
		return nil, err
	}

	// Use optimized query with window functions
	rows, err := d.db.Query(`
		WITH latest_status AS (
			SELECT DISTINCT ON (c.id) c.id, h.success
			FROM checks c
			LEFT JOIN check_history h ON h.check_id = c.id
			WHERE c.enabled = true
			ORDER BY c.id, h.checked_at DESC
		)
		SELECT 
			COUNT(*) FILTER (WHERE success = true) as up_count,
			COUNT(*) FILTER (WHERE success = false OR success IS NULL) as down_count
		FROM latest_status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var upCount, downCount int
		if err := rows.Scan(&upCount, &downCount); err != nil {
			return nil, err
		}
		stats.UpChecks = upCount
		stats.DownChecks = downCount
	}

	var totalChecks, successfulChecks int64
	uptimeQuery := `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE h.success = true)
		FROM check_history h
		JOIN checks c ON h.check_id = c.id
		WHERE c.enabled = true`
	uptimeArgs := []interface{}{}
	if since != nil {
		uptimeQuery += " AND h.checked_at >= $1"
		uptimeArgs = append(uptimeArgs, since)
	}
	err = d.db.QueryRow(uptimeQuery, uptimeArgs...).Scan(&totalChecks, &successfulChecks)
	if err == nil && totalChecks > 0 {
		stats.TotalUptime = float64(successfulChecks) / float64(totalChecks) * 100
	}

	return &stats, nil
}

func (d *PostgresDB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (d *PostgresDB) SetSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT(key) DO UPDATE SET value = $2
	`, key, value)
	return err
}

func (d *PostgresDB) GetAllGroups() ([]models.Group, error) {
	rows, err := d.db.Query(`SELECT id, name, sort_order, created_at FROM groups ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (d *PostgresDB) GetGroup(id int64) (*models.Group, error) {
	var g models.Group
	err := d.db.QueryRow(`SELECT id, name, sort_order, created_at FROM groups WHERE id = $1`, id).
		Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (d *PostgresDB) CreateGroup(g *models.Group) error {
	err := d.db.QueryRow(`
		INSERT INTO groups (name, sort_order) VALUES ($1, $2)
		RETURNING id, created_at
	`, g.Name, g.SortOrder).Scan(&g.ID, &g.CreatedAt)
	return err
}

func (d *PostgresDB) UpdateGroup(g *models.Group) error {
	_, err := d.db.Exec(`UPDATE groups SET name = $1, sort_order = $2 WHERE id = $3`, g.Name, g.SortOrder, g.ID)
	return err
}

func (d *PostgresDB) DeleteGroup(id int64) error {
	_, err := d.db.Exec(`DELETE FROM groups WHERE id = $1`, id)
	return err
}

func (d *PostgresDB) GetAllTags() ([]models.Tag, error) {
	rows, err := d.db.Query(`SELECT id, name, color FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (d *PostgresDB) GetTag(id int64) (*models.Tag, error) {
	var t models.Tag
	err := d.db.QueryRow(`SELECT id, name, color FROM tags WHERE id = $1`, id).Scan(&t.ID, &t.Name, &t.Color)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *PostgresDB) CreateTag(t *models.Tag) error {
	err := d.db.QueryRow(`
		INSERT INTO tags (name, color) VALUES ($1, $2)
		RETURNING id
	`, t.Name, t.Color).Scan(&t.ID)
	return err
}

func (d *PostgresDB) UpdateTag(t *models.Tag) error {
	_, err := d.db.Exec(`UPDATE tags SET name = $1, color = $2 WHERE id = $3`, t.Name, t.Color, t.ID)
	return err
}

func (d *PostgresDB) DeleteTag(id int64) error {
	_, err := d.db.Exec(`DELETE FROM tags WHERE id = $1`, id)
	return err
}

func (d *PostgresDB) GetCheckTags(checkID int64) ([]models.Tag, error) {
	rows, err := d.db.Query(`
		SELECT t.id, t.name, t.color 
		FROM tags t 
		JOIN check_tags ct ON t.id = ct.tag_id 
		WHERE ct.check_id = $1
		ORDER BY t.name
	`, checkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (d *PostgresDB) SetCheckTags(checkID int64, tagIDs []int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM check_tags WHERE check_id = $1`, checkID)
	if err != nil {
		return err
	}

	for _, tagID := range tagIDs {
		_, err := tx.Exec(`INSERT INTO check_tags (check_id, tag_id) VALUES ($1, $2)`, checkID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *PostgresDB) GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	err := d.db.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *PostgresDB) GetUserByID(id int64) (*models.User, error) {
	var u models.User
	err := d.db.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *PostgresDB) CreateUser(u *models.User) error {
	err := d.db.QueryRow(`
		INSERT INTO users (username, password_hash) VALUES ($1, $2)
		RETURNING id, created_at
	`, u.Username, u.PasswordHash).Scan(&u.ID, &u.CreatedAt)
	return err
}

func (d *PostgresDB) HasUsers() (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *PostgresDB) CreateAPIKey(key *models.APIKey) error {
	err := d.db.QueryRow(`
		INSERT INTO api_keys (user_id, name, key_hash) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, key.UserID, key.Name, key.KeyHash).Scan(&key.ID, &key.CreatedAt)
	return err
}

func (d *PostgresDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	var k models.APIKey
	var lastUsedAt sql.NullTime
	err := d.db.QueryRow(`SELECT id, user_id, name, key_hash, last_used_at, created_at FROM api_keys WHERE key_hash = $1`, 
		keyHash).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &lastUsedAt, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		k.LastUsedAt = &lastUsedAt.Time
	}
	return &k, nil
}

func (d *PostgresDB) GetAPIKeysByUserID(userID int64) ([]models.APIKey, error) {
	rows, err := d.db.Query(`SELECT id, user_id, name, key_hash, last_used_at, created_at FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, 
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var lastUsedAt sql.NullTime
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &lastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			k.LastUsedAt = &lastUsedAt.Time
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (d *PostgresDB) UpdateAPIKeyLastUsed(id int64) error {
	_, err := d.db.Exec(`UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	return err
}

func (d *PostgresDB) DeleteAPIKey(id int64) error {
	_, err := d.db.Exec(`DELETE FROM api_keys WHERE id = $1`, id)
	return err
}

func (d *PostgresDB) CreateSession(session *models.Session) error {
	err := d.db.QueryRow(`
		INSERT INTO sessions (token, user_id, username, expires_at) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, session.Token, session.UserID, session.Username, session.ExpiresAt).Scan(&session.ID, &session.CreatedAt)
	return err
}

func (d *PostgresDB) GetSessionByToken(token string) (*models.Session, error) {
	var s models.Session
	err := d.db.QueryRow(`SELECT id, token, user_id, username, expires_at, created_at FROM sessions WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP`,
		token).Scan(&s.ID, &s.Token, &s.UserID, &s.Username, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *PostgresDB) DeleteSession(token string) error {
	_, err := d.db.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (d *PostgresDB) DeleteExpiredSessions() error {
	_, err := d.db.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}

func (d *PostgresDB) DeleteUserSessions(userID int64) error {
	_, err := d.db.Exec(`DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (d *PostgresDB) CreateWebAuthnCredential(cred *models.WebAuthnCredential) error {
	err := d.db.QueryRow(`
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, aaguid, sign_count, clone_warning, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, cred.UserID, cred.CredentialID, cred.PublicKey, cred.AttestationType, cred.AAGUID, cred.SignCount, cred.CloneWarning, cred.Name).Scan(&cred.ID, &cred.CreatedAt)
	return err
}

func (d *PostgresDB) GetWebAuthnCredentialsByUserID(userID int64) ([]models.WebAuthnCredential, error) {
	rows, err := d.db.Query(`SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, clone_warning, name, created_at FROM webauthn_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []models.WebAuthnCredential
	for rows.Next() {
		var c models.WebAuthnCredential
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AttestationType, &c.AAGUID, &c.SignCount, &c.CloneWarning, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (d *PostgresDB) GetWebAuthnCredentialByID(credID []byte) (*models.WebAuthnCredential, error) {
	var c models.WebAuthnCredential
	err := d.db.QueryRow(`SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, clone_warning, name, created_at FROM webauthn_credentials WHERE credential_id = $1`, credID).
		Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AttestationType, &c.AAGUID, &c.SignCount, &c.CloneWarning, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *PostgresDB) UpdateWebAuthnCredentialSignCount(credID []byte, signCount uint32) error {
	_, err := d.db.Exec(`UPDATE webauthn_credentials SET sign_count = $1 WHERE credential_id = $2`, signCount, credID)
	return err
}

func (d *PostgresDB) DeleteWebAuthnCredential(id int64) error {
	_, err := d.db.Exec(`DELETE FROM webauthn_credentials WHERE id = $1`, id)
	return err
}
