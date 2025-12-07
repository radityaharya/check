package checker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gocheck/internal/models"
)

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

	value, err := extractJSONValue(jsonData, check.JSONPath)
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
