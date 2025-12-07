package checker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gocheck/internal/db"
	"gocheck/internal/models"
	"gocheck/internal/notifier"
)

type Engine struct {
	db        *db.Database
	notifiers []notifier.Notifier
	checks    map[int64]*checkState
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	broadcast chan *CheckResultEvent
	clients   map[chan *CheckResultEvent]bool
	clientsMu sync.RWMutex
}

type CheckResultEvent struct {
	CheckID       int64                `json:"check_id"`
	Check         models.Check         `json:"check"`
	LastStatus    *models.CheckHistory `json:"last_status"`
	IsUp          bool                 `json:"is_up"`
	LastCheckedAt *time.Time           `json:"last_checked_at"`
}

type checkState struct {
	check      models.Check
	lastStatus *models.CheckHistory
	ticker     *time.Ticker
	stop       chan struct{}
}

func NewEngine(database *db.Database, notifiers []notifier.Notifier) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		db:        database,
		notifiers: notifiers,
		checks:    make(map[int64]*checkState),
		ctx:       ctx,
		cancel:    cancel,
		broadcast: make(chan *CheckResultEvent, 100),
		clients:   make(map[chan *CheckResultEvent]bool),
	}
	go e.broadcaster()
	return e
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

	retries := check.Retries
	if retries < 0 {
		retries = 0
	}
	if retries > 10 {
		retries = 10
	}
	delaySeconds := check.RetryDelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 5
	}
	if delaySeconds > 60 {
		delaySeconds = 60
	}

	var history models.CheckHistory
	for attempt := 0; attempt <= retries; attempt++ {
		h := models.CheckHistory{CheckID: check.ID, CheckedAt: time.Now()}
		start := time.Now()

		switch check.Type {
		case models.CheckTypePing:
			e.performPingCheck(&check, &h, start)
		case models.CheckTypePostgres:
			e.performPostgresCheck(&check, &h, start)
		case models.CheckTypeJSONHTTP:
			e.performJSONHTTPCheck(&check, &h, start)
		case models.CheckTypeDNS:
			e.performDNSCheck(&check, &h, start)
		case models.CheckTypeTailscale:
			e.performTailscaleCheck(&check, &h, start)
		case models.CheckTypeTailscaleService:
			e.performTailscaleServiceCheck(&check, &h, start)
		default:
			e.performHTTPCheck(&check, &h, start)
		}

		history = h
		if history.Success {
			break
		}
		if attempt < retries {
			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
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
	
	// Broadcast the result to SSE clients
	e.broadcastCheckResult(check, &history)
}

func (e *Engine) broadcastCheckResult(check models.Check, history *models.CheckHistory) {
	event := &CheckResultEvent{
		CheckID:       check.ID,
		Check:         check,
		LastStatus:    history,
		IsUp:          history.Success,
		LastCheckedAt: &history.CheckedAt,
	}
	
	// Non-blocking send
	select {
	case e.broadcast <- event:
	default:
		// Buffer full, skip this event
	}
}

func (e *Engine) broadcaster() {
	for {
		select {
		case event := <-e.broadcast:
			e.clientsMu.RLock()
			for client := range e.clients {
				select {
				case client <- event:
				default:
					// Client buffer full, skip
				}
			}
			e.clientsMu.RUnlock()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) Subscribe() chan *CheckResultEvent {
	client := make(chan *CheckResultEvent, 10)
	e.clientsMu.Lock()
	e.clients[client] = true
	e.clientsMu.Unlock()
	return client
}

func (e *Engine) Unsubscribe(client chan *CheckResultEvent) {
	e.clientsMu.Lock()
	delete(e.clients, client)
	close(client)
	e.clientsMu.Unlock()
}

func (e *Engine) TriggerCheck(checkID int64) error {
	e.mu.RLock()
	state, exists := e.checks[checkID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("check not found or not enabled")
	}

	go e.performCheck(state)
	return nil
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
	case models.CheckTypeTailscaleService:
		return fmt.Sprintf("Tailscale Service: %s:%d", check.TailscaleServiceHost, check.TailscaleServicePort)
	default:
		return check.URL
	}
}
