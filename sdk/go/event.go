package wt

import "time"

// event mirrors the backend's IngestEventRequest JSON contract. Field order
// and names must match backend/internal/model/event.go exactly.
type event struct {
	Message    string         `json:"message"`
	ErrorType  string         `json:"error_type,omitempty"`
	StackTrace string         `json:"stack_trace,omitempty"`
	Project    string         `json:"project"`
	Tag        string         `json:"tag,omitempty"`
	Context    map[string]any `json:"context,omitempty"`
	Timestamp  *time.Time     `json:"timestamp,omitempty"`
}
