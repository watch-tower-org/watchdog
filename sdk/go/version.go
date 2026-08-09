package wt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// versionBasePath is the backend version endpoint (relative to BaseURL).
const versionBasePath = "/api/watchtower/v1/version"

// expectedProduct identifies the WatchTower backend in version responses.
const expectedProduct = "watchtower"

// VersionInfo describes a WatchTower backend instance.
type VersionInfo struct {
	Product string `json:"product"`
	Version string `json:"version"`
}

// CheckVersion verifies that baseURL points to a WatchTower instance and
// returns its version. The API key is sent for authentication; an invalid or
// revoked key fails the check. The caller controls the timeout via ctx. The
// reported version is informational and may be empty when the instance was
// built without version metadata; only identity (product) is validated.
func CheckVersion(ctx context.Context, baseURL, apiKey string) (VersionInfo, error) {
	var info VersionInfo

	base := strings.TrimRight(baseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+versionBasePath, nil)
	if err != nil {
		return info, fmt.Errorf("watchtower: build version request: %w", err)
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("watchtower: version check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return info, fmt.Errorf("watchtower: invalid or revoked API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return info, fmt.Errorf("watchtower: version check failed: server returned %d", resp.StatusCode)
	}

	// The backend wraps the payload in its standard response envelope, with the
	// version info under "data".
	var payload struct {
		Success bool        `json:"success"`
		Data    VersionInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return info, fmt.Errorf("watchtower: %s is not a WatchTower instance", base)
	}
	if !payload.Success {
		return info, fmt.Errorf("watchtower: %s is not a WatchTower instance", base)
	}
	if payload.Data.Product != expectedProduct {
		return info, fmt.Errorf("watchtower: %s is not a WatchTower instance (product %q)", base, payload.Data.Product)
	}
	return payload.Data, nil
}
