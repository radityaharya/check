package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gocheck/internal/checker"
	"gocheck/internal/db"
	"gocheck/internal/models"
	"gocheck/internal/notifier"
	tailscale "tailscale.com/client/tailscale/v2"
)

type Handlers struct {
	db       *db.Database
	engine   *checker.Engine
	notifier *notifier.DiscordNotifier
}

func NewHandlers(database *db.Database, engine *checker.Engine, discordNotifier *notifier.DiscordNotifier) *Handlers {
	return &Handlers{
		db:       database,
		engine:   engine,
		notifier: discordNotifier,
	}
}

func (h *Handlers) SetNotifier(n *notifier.DiscordNotifier) {
	h.notifier = n
}

func (h *Handlers) GetChecks(w http.ResponseWriter, r *http.Request) {
	checks, err := h.db.GetAllChecks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var checksWithStatus []models.CheckWithStatus
	for _, check := range checks {
		lastStatus, _ := h.db.GetLastStatus(check.ID)
		history, _ := h.db.GetCheckHistory(check.ID, 50)

		cws := models.CheckWithStatus{
			Check:      check,
			LastStatus: lastStatus,
			History:    history,
		}

		if lastStatus != nil {
			cws.IsUp = lastStatus.Success
			cws.LastCheckedAt = &lastStatus.CheckedAt
		}

		checksWithStatus = append(checksWithStatus, cws)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checksWithStatus)
}

func (h *Handlers) CreateCheck(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = models.CheckTypeHTTP
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 60
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 10
	}

	check := models.Check{
		Name:                req.Name,
		Type:                req.Type,
		URL:                 req.URL,
		IntervalSeconds:     req.IntervalSeconds,
		TimeoutSeconds:      req.TimeoutSeconds,
		Enabled:             req.Enabled,
		GroupID:             req.GroupID.Value,
		ExpectedStatusCodes: req.ExpectedStatusCodes,
		Method:              req.Method,
		JSONPath:            req.JSONPath,
		ExpectedJSONValue:   req.ExpectedJSONValue,
		PostgresConnString:  req.PostgresConnString,
		PostgresQuery:       req.PostgresQuery,
		ExpectedQueryValue:  req.ExpectedQueryValue,
		Host:                req.Host,
		DNSHostname:         req.DNSHostname,
		DNSRecordType:       req.DNSRecordType,
		ExpectedDNSValue:    req.ExpectedDNSValue,
		TailscaleDeviceID:   req.TailscaleDeviceID,
	}

	if check.Method == "" {
		check.Method = "GET"
	}
	if len(check.ExpectedStatusCodes) == 0 {
		check.ExpectedStatusCodes = []int{200}
	}
	if check.DNSRecordType == "" && check.Type == models.CheckTypeDNS {
		check.DNSRecordType = "A"
	}

	if err := h.db.CreateCheck(&check); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(req.TagIDs) > 0 {
		h.db.SetCheckTags(check.ID, req.TagIDs)
		check.Tags, _ = h.db.GetCheckTags(check.ID)
	}

	h.engine.AddCheck(check)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(check)
}

func (h *Handlers) UpdateCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	check, err := h.db.GetCheck(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if check == nil {
		http.Error(w, "check not found", http.StatusNotFound)
		return
	}

	var req models.UpdateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		check.Name = *req.Name
	}
	if req.Type != nil {
		check.Type = *req.Type
	}
	if req.URL != nil {
		check.URL = *req.URL
	}
	if req.IntervalSeconds != nil {
		check.IntervalSeconds = *req.IntervalSeconds
	}
	if req.TimeoutSeconds != nil {
		check.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.Enabled != nil {
		check.Enabled = *req.Enabled
	}
	check.GroupID = req.GroupID.Value
	if req.ExpectedStatusCodes != nil {
		check.ExpectedStatusCodes = *req.ExpectedStatusCodes
	}
	if req.Method != nil {
		check.Method = *req.Method
	}
	if req.JSONPath != nil {
		check.JSONPath = *req.JSONPath
	}
	if req.ExpectedJSONValue != nil {
		check.ExpectedJSONValue = *req.ExpectedJSONValue
	}
	if req.PostgresConnString != nil {
		check.PostgresConnString = *req.PostgresConnString
	}
	if req.PostgresQuery != nil {
		check.PostgresQuery = *req.PostgresQuery
	}
	if req.ExpectedQueryValue != nil {
		check.ExpectedQueryValue = *req.ExpectedQueryValue
	}
	if req.Host != nil {
		check.Host = *req.Host
	}
	if req.DNSHostname != nil {
		check.DNSHostname = *req.DNSHostname
	}
	if req.DNSRecordType != nil {
		check.DNSRecordType = *req.DNSRecordType
	}
	if req.ExpectedDNSValue != nil {
		check.ExpectedDNSValue = *req.ExpectedDNSValue
	}
	if req.TailscaleDeviceID != nil {
		check.TailscaleDeviceID = *req.TailscaleDeviceID
	}

	if err := h.db.UpdateCheck(check); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.TagIDs != nil {
		h.db.SetCheckTags(check.ID, *req.TagIDs)
		check.Tags, _ = h.db.GetCheckTags(check.ID)
	}

	h.engine.AddCheck(*check)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}

func (h *Handlers) DeleteCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteCheck(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.engine.RemoveCheck(id)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetCheckHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := h.db.GetCheckHistory(id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	webhookURL, _ := h.db.GetSetting("discord_webhook_url")
	tailscaleAPIKey, _ := h.db.GetSetting("tailscale_api_key")
	tailscaleTailnet, _ := h.db.GetSetting("tailscale_tailnet")

	settings := models.Settings{
		DiscordWebhookURL:  webhookURL,
		TailscaleAPIKey:    tailscaleAPIKey,
		TailscaleTailnet:   tailscaleTailnet,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings models.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.SetSetting("discord_webhook_url", settings.DiscordWebhookURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("tailscale_api_key", settings.TailscaleAPIKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.db.SetSetting("tailscale_tailnet", settings.TailscaleTailnet); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.notifier = notifier.NewDiscordNotifier(settings.DiscordWebhookURL)
	h.engine.UpdateNotifier(settings.DiscordWebhookURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *Handlers) TestWebhook(w http.ResponseWriter, r *http.Request) {
	if h.notifier == nil {
		http.Error(w, "notifier not configured", http.StatusBadRequest)
		return
	}

	if err := h.notifier.TestWebhook(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Test notification sent successfully"})
}

func (h *Handlers) GetTailscaleDevices(w http.ResponseWriter, r *http.Request) {
	apiKey, _ := h.db.GetSetting("tailscale_api_key")
	tailnet, _ := h.db.GetSetting("tailscale_tailnet")

	if apiKey == "" || tailnet == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tailscale API key or tailnet not configured"})
		return
	}

	client := &tailscale.Client{
		Tailnet: tailnet,
		APIKey:  apiKey,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	devices, err := client.Devices().List(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	type DeviceInfo struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Hostname  string   `json:"hostname"`
		Addresses []string `json:"addresses"`
		Online    bool     `json:"online"`
		OS        string   `json:"os"`
		LastSeen  string   `json:"last_seen"`
	}

	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		online := d.ConnectedToControl
		if !online && !d.LastSeen.Time.IsZero() {
			online = time.Since(d.LastSeen.Time) < 5*time.Minute
		}
		lastSeen := ""
		if !d.LastSeen.Time.IsZero() {
			lastSeen = d.LastSeen.Time.Format(time.RFC3339)
		}
		result = append(result, DeviceInfo{
			ID:        d.ID,
			Name:      d.Name,
			Hostname:  d.Hostname,
			Addresses: d.Addresses,
			Online:    online,
			OS:        d.OS,
			LastSeen:  lastSeen,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) TestTailscale(w http.ResponseWriter, r *http.Request) {
	apiKey, _ := h.db.GetSetting("tailscale_api_key")
	tailnet, _ := h.db.GetSetting("tailscale_tailnet")

	if apiKey == "" || tailnet == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tailscale API key or tailnet not configured"})
		return
	}

	client := &tailscale.Client{
		Tailnet: tailnet,
		APIKey:  apiKey,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	devices, err := client.Devices().List(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "ok",
		"message":      "Connected to Tailscale successfully",
		"device_count": len(devices),
	})
}

func (h *Handlers) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.db.GetAllGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []models.Group{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func (h *Handlers) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	group := models.Group{Name: req.Name, SortOrder: req.SortOrder}
	if err := h.db.CreateGroup(&group); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (h *Handlers) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	group, err := h.db.GetGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if group == nil {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}

	var req models.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}

	if err := h.db.UpdateGroup(group); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func (h *Handlers) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteGroup(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.db.GetAllTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tags == nil {
		tags = []models.Tag{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *Handlers) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Color == "" {
		req.Color = "#6b7280"
	}

	tag := models.Tag{Name: req.Name, Color: req.Color}
	if err := h.db.CreateTag(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (h *Handlers) UpdateTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	tag, err := h.db.GetTag(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag == nil {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}

	var req models.UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		tag.Name = *req.Name
	}
	if req.Color != nil {
		tag.Color = *req.Color
	}

	if err := h.db.UpdateTag(tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (h *Handlers) DeleteTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteTag(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetGroupedChecks(w http.ResponseWriter, r *http.Request) {
	checks, err := h.db.GetAllChecks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groups, err := h.db.GetAllGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupMap := make(map[int64]*models.GroupWithChecks)
	for _, g := range groups {
		groupMap[g.ID] = &models.GroupWithChecks{
			Group:  g,
			Checks: []models.CheckWithStatus{},
			IsUp:   true,
		}
	}

	ungrouped := &models.GroupWithChecks{
		Group:  models.Group{ID: 0, Name: "Ungrouped"},
		Checks: []models.CheckWithStatus{},
		IsUp:   true,
	}

	for _, check := range checks {
		lastStatus, _ := h.db.GetLastStatus(check.ID)
		history, _ := h.db.GetCheckHistory(check.ID, 50)

		cws := models.CheckWithStatus{
			Check:      check,
			LastStatus: lastStatus,
			History:    history,
		}

		if lastStatus != nil {
			cws.IsUp = lastStatus.Success
			cws.LastCheckedAt = &lastStatus.CheckedAt
		}

		if check.GroupID != nil {
			if g, ok := groupMap[*check.GroupID]; ok {
				g.Checks = append(g.Checks, cws)
				// Only count enabled checks towards group status
				if check.Enabled {
					if !cws.IsUp {
						g.IsUp = false
						g.DownCount++
					} else {
						g.UpCount++
					}
				}
			} else {
				ungrouped.Checks = append(ungrouped.Checks, cws)
				if check.Enabled {
					if !cws.IsUp {
						ungrouped.IsUp = false
						ungrouped.DownCount++
					} else {
						ungrouped.UpCount++
					}
				}
			}
		} else {
			ungrouped.Checks = append(ungrouped.Checks, cws)
			if check.Enabled {
				if !cws.IsUp {
					ungrouped.IsUp = false
					ungrouped.DownCount++
				} else {
					ungrouped.UpCount++
				}
			}
		}
	}

	result := []models.GroupWithChecks{}
	for _, g := range groups {
		if gwc, ok := groupMap[g.ID]; ok {
			result = append(result, *gwc)
		}
	}
	if len(ungrouped.Checks) > 0 {
		result = append(result, *ungrouped)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
