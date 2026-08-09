package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	v "github.com/watch-tower-org/watchdog/backend/internal/version"
)

func TestVersionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewController())
	r := gin.New()
	r.GET("/version", h.Version)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Product string `json:"product"`
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body.Success {
		t.Error("success = false, want true")
	}
	if body.Data.Product != "watchdog" {
		t.Errorf("product = %q, want watchdog", body.Data.Product)
	}
	if body.Data.Version != v.Version {
		t.Errorf("version = %q, want %q", body.Data.Version, v.Version)
	}
}
