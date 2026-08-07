package web

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, "/api/watchtower/v1")

	cases := []struct {
		path string
		code int
		body string
	}{
		{"/", 200, "<!doctype html>"},
		{"/index.html", 200, "<!doctype html>"},
		{"/dashboard", 200, "<!doctype html>"},
		{"/settings", 200, "<!doctype html>"},
		{"/login", 200, "<!doctype html>"},
		{"/assets/missing.js", 200, "<!doctype html>"},
		{"/api/watchtower/v1/nope", 404, "{\"code\":404"},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		t.Logf("GET %s -> %d loc=%q body=%.40q", tc.path, w.Code, w.Header().Get("Location"), w.Body.String())
		if w.Code != tc.code {
			t.Errorf("GET %s: got %d, want %d", tc.path, w.Code, tc.code)
		}
		if w.Code == 200 && len(w.Body.String()) < 10 {
			t.Errorf("GET %s: expected html body, got %q", tc.path, w.Body.String())
		}
	}
}
