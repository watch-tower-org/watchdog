package uptime

import (
	"io"
	"net/http"
	"time"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

// CheckResult describes the outcome of a single HTTP health check.
type CheckResult struct {
	Status         model.ServiceStatus `json:"status"`
	StatusCode     int                 `json:"status_code"`
	ResponseTimeMs int                 `json:"response_time_ms"`
	Error          *string             `json:"error"`
}

// CheckNowResult is the payload returned by the manual "Check now" endpoint.
type CheckNowResult struct {
	Service *model.MonitoredService `json:"service"`
	Result  CheckResult             `json:"result"`
}

// runCheck performs a GET request against the service URL and determines
// up/down. Any 2xx/3xx status is "up"; network errors, timeouts and 4xx/5xx
// responses are "down".
func runCheck(url string, timeout time.Duration) *CheckResult {
	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Get(url)
	elapsed := int(time.Since(start) / time.Millisecond)

	if err != nil {
		msg := err.Error()
		return &CheckResult{
			Status:         model.ServiceStatusDown,
			ResponseTimeMs: elapsed,
			Error:          &msg,
		}
	}
	defer resp.Body.Close()

	// Drain a small amount so the connection can be reused; never read the body.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	status := model.ServiceStatusDown
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = model.ServiceStatusUp
	}

	return &CheckResult{
		Status:         status,
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: elapsed,
	}
}
