package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gocheck/internal/models"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	dsn := dbPath + "?_foreign_keys=1&_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for SQLite
	// Allow multiple readers but serialize writes via WAL mode
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	d := &SQLiteDB{db: db}
	if err := d.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

func (d *SQLiteDB) Close() error {
	return d.db.Close()
}

func (d *SQLiteDB) initSchema() error {
	checksTable := `
	CREATE TABLE IF NOT EXISTS checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'http',
		url TEXT,
		interval_seconds INTEGER NOT NULL DEFAULT 60,
		timeout_seconds INTEGER NOT NULL DEFAULT 10,
		retries INTEGER NOT NULL DEFAULT 0,
		retry_delay_seconds INTEGER NOT NULL DEFAULT 5,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expected_status_codes TEXT,
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
		group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL
	);`

	historyTable := `
	CREATE TABLE IF NOT EXISTS check_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		check_id INTEGER NOT NULL,
		status_code INTEGER,
		response_time_ms INTEGER,
		success BOOLEAN NOT NULL,
		error_message TEXT,
		checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		response_body TEXT,
		FOREIGN KEY (check_id) REFERENCES checks(id) ON DELETE CASCADE
	);`

	settingsTable := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`

	groupsTable := `
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	tagsTable := `
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#6b7280'
	);`

	checkTagsTable := `
	CREATE TABLE IF NOT EXISTS check_tags (
		check_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (check_id, tag_id),
		FOREIGN KEY (check_id) REFERENCES checks(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);`

	indexes := `
	CREATE INDEX IF NOT EXISTS idx_check_history_check_id ON check_history(check_id);
	CREATE INDEX IF NOT EXISTS idx_check_history_checked_at ON check_history(checked_at);
	CREATE INDEX IF NOT EXISTS idx_check_history_check_id_checked_at ON check_history(check_id, checked_at DESC);
	CREATE INDEX IF NOT EXISTS idx_checks_enabled ON checks(enabled);
	CREATE INDEX IF NOT EXISTS idx_checks_created_at ON checks(created_at);
	CREATE INDEX IF NOT EXISTS idx_checks_group_id ON checks(group_id);
	`

	if _, err := d.db.Exec(checksTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(historyTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(settingsTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(groupsTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(tagsTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(checkTagsTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(indexes); err != nil {
		return err
	}

	d.migrateSchema()

	return nil
}

func (d *SQLiteDB) migrateSchema() {
	columns := []struct {
		name         string
		defaultValue string
	}{
		{"type", "'http'"},
		{"expected_status_codes", "'[200]'"},
		{"method", "'GET'"},
		{"json_path", "NULL"},
		{"expected_json_value", "NULL"},
		{"postgres_conn_string", "NULL"},
		{"postgres_query", "NULL"},
		{"expected_query_value", "NULL"},
		{"host", "NULL"},
		{"dns_hostname", "NULL"},
		{"dns_record_type", "NULL"},
		{"expected_dns_value", "NULL"},
		{"group_id", "NULL"},
		{"tailscale_device_id", "NULL"},
		{"retries", "0"},
		{"retry_delay_seconds", "5"},
	}

	for _, col := range columns {
		query := fmt.Sprintf("ALTER TABLE checks ADD COLUMN %s TEXT DEFAULT %s", col.name, col.defaultValue)
		d.db.Exec(query)
	}

	d.db.Exec("ALTER TABLE check_history ADD COLUMN response_body TEXT")

	d.db.Exec(`
		UPDATE checks 
		SET expected_status_codes = '[' || COALESCE(expected_status_code, 200) || ']'
		WHERE expected_status_codes IS NULL AND expected_status_code IS NOT NULL
	`)
}

func (d *SQLiteDB) parseStatusCodes(s string) []int {
	if s == "" {
		return []int{200}
	}
	var codes []int
	if err := json.Unmarshal([]byte(s), &codes); err != nil {
		return []int{200}
	}
	return codes
}

func (d *SQLiteDB) encodeStatusCodes(codes []int) string {
	if len(codes) == 0 {
		return "[200]"
	}
	data, _ := json.Marshal(codes)
	return string(data)
}

func (d *SQLiteDB) GetAllChecks() ([]models.Check, error) {
	rows, err := d.db.Query(`
		SELECT id, name, COALESCE(type, 'http'), COALESCE(url, ''), interval_seconds, timeout_seconds, COALESCE(retries, 0), COALESCE(retry_delay_seconds, 5), enabled, created_at,
			COALESCE(expected_status_codes, '[200]'), COALESCE(method, 'GET'), COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), COALESCE(host, ''),
			COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, '')
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
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, &c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
			&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
			&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
			&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID); err != nil {
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

func (d *SQLiteDB) GetCheck(id int64) (*models.Check, error) {
	var c models.Check
	var statusCodesJSON string
	var groupID sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, name, COALESCE(type, 'http'), COALESCE(url, ''), interval_seconds, timeout_seconds, COALESCE(retries, 0), COALESCE(retry_delay_seconds, 5), enabled, created_at,
			COALESCE(expected_status_codes, '[200]'), COALESCE(method, 'GET'), COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), COALESCE(host, ''),
			COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, '')
		FROM checks
		WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, &c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
		&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
		&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
		&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID)

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

func (d *SQLiteDB) CreateCheck(c *models.Check) error {
	statusCodesJSON := d.encodeStatusCodes(c.ExpectedStatusCodes)
	result, err := d.db.Exec(`
		INSERT INTO checks (name, type, url, interval_seconds, timeout_seconds, retries, retry_delay_seconds, enabled,
			expected_status_codes, method, json_path, expected_json_value,
			postgres_conn_string, postgres_query, expected_query_value, host,
			dns_hostname, dns_record_type, expected_dns_value, group_id, tailscale_device_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.Name, c.Type, c.URL, c.IntervalSeconds, c.TimeoutSeconds, c.Retries, c.RetryDelaySeconds, c.Enabled,
		statusCodesJSON, c.Method, c.JSONPath, c.ExpectedJSONValue,
		c.PostgresConnString, c.PostgresQuery, c.ExpectedQueryValue, c.Host,
		c.DNSHostname, c.DNSRecordType, c.ExpectedDNSValue, c.GroupID, c.TailscaleDeviceID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	c.CreatedAt = time.Now()

	return nil
}

func (d *SQLiteDB) UpdateCheck(c *models.Check) error {
	statusCodesJSON := d.encodeStatusCodes(c.ExpectedStatusCodes)
	_, err := d.db.Exec(`
		UPDATE checks
		SET name = ?, type = ?, url = ?, interval_seconds = ?, timeout_seconds = ?, retries = ?, retry_delay_seconds = ?, enabled = ?,
			expected_status_codes = ?, method = ?, json_path = ?, expected_json_value = ?,
			postgres_conn_string = ?, postgres_query = ?, expected_query_value = ?, host = ?,
			dns_hostname = ?, dns_record_type = ?, expected_dns_value = ?, group_id = ?, tailscale_device_id = ?
		WHERE id = ?
	`, c.Name, c.Type, c.URL, c.IntervalSeconds, c.TimeoutSeconds, c.Retries, c.RetryDelaySeconds, c.Enabled,
		statusCodesJSON, c.Method, c.JSONPath, c.ExpectedJSONValue,
		c.PostgresConnString, c.PostgresQuery, c.ExpectedQueryValue, c.Host,
		c.DNSHostname, c.DNSRecordType, c.ExpectedDNSValue, c.GroupID, c.TailscaleDeviceID, c.ID)
	return err
}

func (d *SQLiteDB) DeleteCheck(id int64) error {
	_, err := d.db.Exec("DELETE FROM checks WHERE id = ?", id)
	return err
}

func (d *SQLiteDB) GetEnabledChecks() ([]models.Check, error) {
	rows, err := d.db.Query(`
		SELECT id, name, COALESCE(type, 'http'), COALESCE(url, ''), interval_seconds, timeout_seconds, COALESCE(retries, 0), COALESCE(retry_delay_seconds, 5), enabled, created_at,
			COALESCE(expected_status_codes, '[200]'), COALESCE(method, 'GET'), COALESCE(json_path, ''), COALESCE(expected_json_value, ''),
			COALESCE(postgres_conn_string, ''), COALESCE(postgres_query, ''), COALESCE(expected_query_value, ''), COALESCE(host, ''),
			COALESCE(dns_hostname, ''), COALESCE(dns_record_type, ''), COALESCE(expected_dns_value, ''), group_id, COALESCE(tailscale_device_id, '')
		FROM checks
		WHERE enabled = 1
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
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.IntervalSeconds, &c.TimeoutSeconds, &c.Retries, &c.RetryDelaySeconds, &c.Enabled, &c.CreatedAt,
			&statusCodesJSON, &c.Method, &c.JSONPath, &c.ExpectedJSONValue,
			&c.PostgresConnString, &c.PostgresQuery, &c.ExpectedQueryValue, &c.Host,
			&c.DNSHostname, &c.DNSRecordType, &c.ExpectedDNSValue, &groupID, &c.TailscaleDeviceID); err != nil {
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

func (d *SQLiteDB) AddHistory(h *models.CheckHistory) error {
	_, err := d.db.Exec(`
		INSERT INTO check_history (check_id, status_code, response_time_ms, success, error_message, response_body)
		VALUES (?, ?, ?, ?, ?, ?)
	`, h.CheckID, h.StatusCode, h.ResponseTimeMs, h.Success, h.ErrorMessage, h.ResponseBody)
	return err
}

func (d *SQLiteDB) GetCheckHistory(checkID int64, since *time.Time, limit int) ([]models.CheckHistory, error) {
	query := `
		SELECT id, check_id, status_code, response_time_ms, success, COALESCE(error_message, ''), checked_at
		FROM check_history
		WHERE check_id = ?`
	args := []interface{}{checkID}

	if since != nil {
		query += " AND checked_at >= ?"
		args = append(args, since.UTC().Format("2006-01-02 15:04:05"))
	}

	query += " ORDER BY checked_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
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

func (d *SQLiteDB) GetCheckHistoryAggregated(checkID int64, since *time.Time, bucketMinutes int, limit int) ([]models.CheckHistory, error) {
	// Aggregate data into time buckets for large ranges
	query := `
		SELECT 
			MAX(id) as id,
			check_id,
			CAST(AVG(status_code) AS INTEGER) as status_code,
			CAST(AVG(response_time_ms) AS INTEGER) as response_time_ms,
			MIN(CAST(success AS INTEGER)) as success,
			'' as error_message,
			datetime((strftime('%s', checked_at) / (? * 60)) * (? * 60), 'unixepoch') as checked_at
		FROM check_history
		WHERE check_id = ?`
	args := []interface{}{bucketMinutes, bucketMinutes, checkID}

	if since != nil {
		query += " AND checked_at >= ?"
		args = append(args, since.UTC().Format("2006-01-02 15:04:05"))
	}

	query += " GROUP BY datetime((strftime('%s', checked_at) / (? * 60)) * (? * 60), 'unixepoch') ORDER BY checked_at DESC"
	args = append(args, bucketMinutes, bucketMinutes)

	if limit > 0 {
		query += " LIMIT ?"
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

func (d *SQLiteDB) GetLastStatus(checkID int64) (*models.CheckHistory, error) {
	var h models.CheckHistory
	err := d.db.QueryRow(`
		SELECT id, check_id, status_code, response_time_ms, success, COALESCE(error_message, ''), checked_at
		FROM check_history
		WHERE check_id = ?
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

func (d *SQLiteDB) GetStats(since *time.Time) (*models.Stats, error) {
	var stats models.Stats

	err := d.db.QueryRow("SELECT COUNT(*) FROM checks").Scan(&stats.TotalChecks)
	if err != nil {
		return nil, err
	}

	err = d.db.QueryRow("SELECT COUNT(*) FROM checks WHERE enabled = 1").Scan(&stats.ActiveChecks)
	if err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT DISTINCT c.id,
			(SELECT success FROM check_history WHERE check_id = c.id ORDER BY checked_at DESC LIMIT 1) as last_success
		FROM checks c
		WHERE enabled = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var upCount, downCount int
	for rows.Next() {
		var checkID int64
		var lastSuccess sql.NullBool
		if err := rows.Scan(&checkID, &lastSuccess); err != nil {
			continue
		}
		if lastSuccess.Valid && lastSuccess.Bool {
			upCount++
		} else {
			downCount++
		}
	}
	stats.UpChecks = upCount
	stats.DownChecks = downCount

	var totalChecks, successfulChecks int64
	uptimeQuery := `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN h.success = 1 THEN 1 ELSE 0 END), 0)
		FROM check_history h
		JOIN checks c ON h.check_id = c.id
		WHERE c.enabled = 1`
	uptimeArgs := []interface{}{}
	if since != nil {
		uptimeQuery += " AND h.checked_at >= ?"
		uptimeArgs = append(uptimeArgs, since.UTC().Format("2006-01-02 15:04:05"))
	}
	err = d.db.QueryRow(uptimeQuery, uptimeArgs...).Scan(&totalChecks, &successfulChecks)
	if err == nil && totalChecks > 0 {
		stats.TotalUptime = float64(successfulChecks) / float64(totalChecks) * 100
	}

	return &stats, nil
}

func (d *SQLiteDB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (d *SQLiteDB) SetSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (d *SQLiteDB) GetAllGroups() ([]models.Group, error) {
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

func (d *SQLiteDB) GetGroup(id int64) (*models.Group, error) {
	var g models.Group
	err := d.db.QueryRow(`SELECT id, name, sort_order, created_at FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (d *SQLiteDB) CreateGroup(g *models.Group) error {
	result, err := d.db.Exec(`INSERT INTO groups (name, sort_order) VALUES (?, ?)`, g.Name, g.SortOrder)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	g.ID = id
	return nil
}

func (d *SQLiteDB) UpdateGroup(g *models.Group) error {
	_, err := d.db.Exec(`UPDATE groups SET name = ?, sort_order = ? WHERE id = ?`, g.Name, g.SortOrder, g.ID)
	return err
}

func (d *SQLiteDB) DeleteGroup(id int64) error {
	d.db.Exec(`UPDATE checks SET group_id = NULL WHERE group_id = ?`, id)
	_, err := d.db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	return err
}

func (d *SQLiteDB) GetAllTags() ([]models.Tag, error) {
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

func (d *SQLiteDB) GetTag(id int64) (*models.Tag, error) {
	var t models.Tag
	err := d.db.QueryRow(`SELECT id, name, color FROM tags WHERE id = ?`, id).Scan(&t.ID, &t.Name, &t.Color)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *SQLiteDB) CreateTag(t *models.Tag) error {
	result, err := d.db.Exec(`INSERT INTO tags (name, color) VALUES (?, ?)`, t.Name, t.Color)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}

func (d *SQLiteDB) UpdateTag(t *models.Tag) error {
	_, err := d.db.Exec(`UPDATE tags SET name = ?, color = ? WHERE id = ?`, t.Name, t.Color, t.ID)
	return err
}

func (d *SQLiteDB) DeleteTag(id int64) error {
	d.db.Exec(`DELETE FROM check_tags WHERE tag_id = ?`, id)
	_, err := d.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	return err
}

func (d *SQLiteDB) GetCheckTags(checkID int64) ([]models.Tag, error) {
	rows, err := d.db.Query(`
		SELECT t.id, t.name, t.color 
		FROM tags t 
		JOIN check_tags ct ON t.id = ct.tag_id 
		WHERE ct.check_id = ?
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

func (d *SQLiteDB) SetCheckTags(checkID int64, tagIDs []int64) error {
	d.db.Exec(`DELETE FROM check_tags WHERE check_id = ?`, checkID)
	for _, tagID := range tagIDs {
		_, err := d.db.Exec(`INSERT INTO check_tags (check_id, tag_id) VALUES (?, ?)`, checkID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}
