package checker

import (
	"fmt"
	"net/http"
	"time"

	"gocheck/internal/models"
)

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
