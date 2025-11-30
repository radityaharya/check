package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"gocheck/internal/db"
	"gocheck/internal/models"
	"gocheck/internal/notifier"
	tailscale "tailscale.com/client/tailscale/v2"
)

type Engine struct {
	db        *db.Database
	notifiers []notifier.Notifier
	checks    map[int64]*checkState
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type checkState struct {
	check      models.Check
	lastStatus *models.CheckHistory
	ticker     *time.Ticker
	stop       chan struct{}
}

func NewEngine(database *db.Database, notifiers []notifier.Notifier) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		db:        database,
		notifiers: notifiers,
		checks:    make(map[int64]*checkState),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (e *Engine) Start() error {
	checks, err := e.db.GetEnabledChecks()
	if err != nil {
		return fmt.Errorf("failed to load checks: %w", err)
	}

	for _, check := range checks {
		e.addCheck(check)
	}

	return nil
}

func (e *Engine) Stop() {
	e.cancel()
	e.mu.Lock()
	for _, state := range e.checks {
		close(state.stop)
		state.ticker.Stop()
	}
	e.mu.Unlock()
	e.wg.Wait()
}

func (e *Engine) UpdateNotifiers(notifiers []notifier.Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers = notifiers
}

func (e *Engine) AddCheck(check models.Check) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if check.Enabled {
		e.addCheck(check)
	} else {
		e.removeCheck(check.ID)
	}
}

func (e *Engine) RemoveCheck(checkID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.removeCheck(checkID)
}

func (e *Engine) addCheck(check models.Check) {
	if state, exists := e.checks[check.ID]; exists {
		close(state.stop)
		state.ticker.Stop()
	}

	lastStatus, _ := e.db.GetLastStatus(check.ID)

	state := &checkState{
		check:      check,
		lastStatus: lastStatus,
		ticker:     time.NewTicker(time.Duration(check.IntervalSeconds) * time.Second),
		stop:       make(chan struct{}),
	}

	e.checks[check.ID] = state

	e.wg.Add(1)
	go e.runCheck(state)
}

func (e *Engine) removeCheck(checkID int64) {
	if state, exists := e.checks[checkID]; exists {
		close(state.stop)
		state.ticker.Stop()
		delete(e.checks, checkID)
	}
}

func (e *Engine) runCheck(state *checkState) {
	defer e.wg.Done()

	e.performCheck(state)

	for {
		select {
		case <-state.ticker.C:
			e.performCheck(state)
		case <-state.stop:
			return
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) performCheck(state *checkState) {
	check := state.check
	start := time.Now()

	var history models.CheckHistory
	history.CheckID = check.ID
	history.CheckedAt = time.Now()

	switch check.Type {
	case models.CheckTypePing:
		e.performPingCheck(&check, &history, start)
	case models.CheckTypePostgres:
		e.performPostgresCheck(&check, &history, start)
	case models.CheckTypeJSONHTTP:
		e.performJSONHTTPCheck(&check, &history, start)
	case models.CheckTypeDNS:
		e.performDNSCheck(&check, &history, start)
	case models.CheckTypeTailscale:
		e.performTailscaleCheck(&check, &history, start)
	default:
		e.performHTTPCheck(&check, &history, start)
	}

	e.db.AddHistory(&history)

	statusChanged := false
	if state.lastStatus == nil {
		statusChanged = true
	} else {
		statusChanged = state.lastStatus.Success != history.Success
	}

	if statusChanged {
		e.mu.RLock()
		notifiers := e.notifiers
		e.mu.RUnlock()
		for _, n := range notifiers {
			if n != nil {
				n.SendStatusChange(
					check.Name,
					e.getCheckTarget(check),
					history.Success,
					history.StatusCode,
					history.ResponseTimeMs,
					history.ErrorMessage,
				)
			}
		}
	}

	state.lastStatus = &history
}

func (e *Engine) getCheckTarget(check models.Check) string {
	switch check.Type {
	case models.CheckTypePing:
		return check.Host
	case models.CheckTypePostgres:
		return "PostgreSQL: " + check.Name
	case models.CheckTypeDNS:
		return check.DNSHostname
	case models.CheckTypeTailscale:
		return "Tailscale: " + check.TailscaleDeviceID
	default:
		return check.URL
	}
}

func (e *Engine) performHTTPCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	client := &http.Client{
		Timeout: time.Duration(check.TimeoutSeconds) * time.Second,
	}

	method := check.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, check.URL, nil)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("invalid request: %v", err)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	resp, err := client.Do(req)
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

	if !success {
		// Fallback to 2xx range if no specific codes match
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			success = true
		}
	}

	if success {
		history.Success = true
	} else {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("unexpected status code: %d (expected: %v)", resp.StatusCode, expectedStatusCodes)
	}
}

func (e *Engine) performPingCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	host := check.Host
	if host == "" {
		history.Success = false
		history.ErrorMessage = "no host specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	timeout := time.Duration(check.TimeoutSeconds) * time.Second

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", check.TimeoutSeconds*1000), host)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", check.TimeoutSeconds), host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("ping failed: %v", err)
		return
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "time=") || strings.Contains(outputStr, "Time=") {
		re := regexp.MustCompile(`time[=<](\d+\.?\d*)`)
		matches := re.FindStringSubmatch(outputStr)
		if len(matches) > 1 {
			history.Success = true
		} else {
			history.Success = true
		}
	} else if strings.Contains(outputStr, "bytes from") || strings.Contains(outputStr, "Reply from") {
		history.Success = true
	} else {
		history.Success = false
		history.ErrorMessage = "no response from host"
	}
}

func (e *Engine) performPostgresCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.PostgresConnString == "" {
		history.Success = false
		history.ErrorMessage = "no connection string specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", check.PostgresConnString)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("connection error: %v", err)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Duration(check.TimeoutSeconds) * time.Second)
	db.SetMaxOpenConns(1)

	if check.PostgresQuery == "" {
		err = db.PingContext(ctx)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		if err != nil {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("ping failed: %v", err)
			return
		}
		history.Success = true
		return
	}

	var result string
	err = db.QueryRowContext(ctx, check.PostgresQuery).Scan(&result)
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("query failed: %v", err)
		return
	}

	history.ResponseBody = result

	if check.ExpectedQueryValue != "" {
		if result == check.ExpectedQueryValue {
			history.Success = true
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("expected '%s', got '%s'", check.ExpectedQueryValue, result)
		}
	} else {
		history.Success = true
	}
}

func (e *Engine) performJSONHTTPCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	client := &http.Client{
		Timeout: time.Duration(check.TimeoutSeconds) * time.Second,
	}

	method := check.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, check.URL, nil)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("invalid request: %v", err)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = err.Error()
		history.StatusCode = 0
		return
	}
	defer resp.Body.Close()

	history.StatusCode = resp.StatusCode

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("failed to read body: %v", err)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		return
	}

	if check.JSONPath == "" {
		history.Success = true
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("invalid JSON: %v", err)
		return
	}

	value, err := e.extractJSONValue(jsonData, check.JSONPath)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("JSON path error: %v", err)
		return
	}

	history.ResponseBody = fmt.Sprintf("%v", value)

	if check.ExpectedJSONValue != "" {
		valueStr := fmt.Sprintf("%v", value)
		if valueStr == check.ExpectedJSONValue {
			history.Success = true
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("expected '%s', got '%s'", check.ExpectedJSONValue, valueStr)
		}
	} else {
		history.Success = true
	}
}

func (e *Engine) performDNSCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.DNSHostname == "" {
		history.Success = false
		history.ErrorMessage = "no hostname specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	recordType := check.DNSRecordType
	if recordType == "" {
		recordType = "A"
	}

	resolver := &net.Resolver{
		PreferGo: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	var records []string
	var err error

	switch strings.ToUpper(recordType) {
	case "A":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip4", check.DNSHostname)
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "AAAA":
		var ips []net.IP
		ips, err = resolver.LookupIP(ctx, "ip6", check.DNSHostname)
		for _, ip := range ips {
			records = append(records, ip.String())
		}
	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(ctx, check.DNSHostname)
		if err == nil {
			records = append(records, cname)
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(ctx, check.DNSHostname)
		if err == nil {
			for _, mx := range mxs {
				records = append(records, fmt.Sprintf("%s (priority: %d)", mx.Host, mx.Pref))
			}
		}
	case "TXT":
		var txts []string
		txts, err = resolver.LookupTXT(ctx, check.DNSHostname)
		if err == nil {
			records = txts
		}
	default:
		err = fmt.Errorf("unsupported record type: %s", recordType)
	}

	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("DNS lookup failed: %v", err)
		return
	}

	if len(records) == 0 {
		history.Success = false
		history.ErrorMessage = "no records found"
		return
	}

	history.ResponseBody = strings.Join(records, ", ")

	if check.ExpectedDNSValue != "" {
		found := false
		for _, record := range records {
			if record == check.ExpectedDNSValue || strings.Contains(record, check.ExpectedDNSValue) {
				found = true
				break
			}
		}
		if found {
			history.Success = true
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("expected value '%s' not found in records: %v", check.ExpectedDNSValue, records)
		}
	} else {
		history.Success = true
	}
}

func (e *Engine) extractJSONValue(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found", part)
			}
		case []interface{}:
			var idx int
			if _, err := fmt.Sscanf(part, "[%d]", &idx); err == nil {
				if idx < 0 || idx >= len(v) {
					return nil, fmt.Errorf("index %d out of range", idx)
				}
				current = v[idx]
			} else {
				return nil, fmt.Errorf("expected array index, got '%s'", part)
			}
		default:
			return nil, fmt.Errorf("cannot navigate into %T", v)
		}
	}

	return current, nil
}

func (e *Engine) CheckConnectivity(checkType models.CheckType, target string, timeout int) (bool, string, int) {
	start := time.Now()

	switch checkType {
	case models.CheckTypePing:
		conn, err := net.DialTimeout("ip4:icmp", target, time.Duration(timeout)*time.Second)
		responseTime := int(time.Since(start).Milliseconds())
		if err != nil {
			return false, err.Error(), responseTime
		}
		conn.Close()
		return true, "", responseTime
	default:
		return false, "unknown check type", 0
	}
}

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
