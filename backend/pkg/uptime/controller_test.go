package uptime

import (
	"testing"
)

func TestValidateInput(t *testing.T) {
	cases := []struct {
		name                string
		displayName         string
		rawURL              string
		intervalSeconds     int
		timeoutSeconds      int
		failuresBeforeAlert int
		wantErr             bool
	}{
		{"valid", "payments", "https://payments.example/healthz", 60, 10, 1, false},
		{"missing name", "", "https://example.com", 60, 10, 1, true},
		{"missing url", "payments", "", 60, 10, 1, true},
		{"bad scheme", "payments", "ftp://example.com", 60, 10, 1, true},
		{"not a url", "payments", "not-a-url", 60, 10, 1, true},
		{"no host", "payments", "https://", 60, 10, 1, true},
		{"interval too small", "payments", "https://example.com", 5, 1, 1, true},
		{"timeout zero", "payments", "https://example.com", 60, 0, 1, true},
		{"timeout equals interval", "payments", "https://example.com", 60, 60, 1, true},
		{"timeout exceeds interval", "payments", "https://example.com", 30, 60, 1, true},
		{"failures zero", "payments", "https://example.com", 60, 10, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInput(tc.displayName, tc.rawURL, tc.intervalSeconds, tc.timeoutSeconds, tc.failuresBeforeAlert)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateInput = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
