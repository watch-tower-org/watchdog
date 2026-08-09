package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
	"github.com/watch-tower-org/watchtower/backend/internal/validator"
)

type Handler struct {
	controller *Controller
}

func NewHandler(controller *Controller) *Handler {
	return &Handler{controller: controller}
}

// Ingest accepts a single event object or an array of events, validates each,
// enforces API-key project scoping, and stores them.
func (h *Handler) Ingest(c *gin.Context) {
	reqs, err := bindEvents(c)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}
	if len(reqs) == 0 {
		res.BadRequest(c, "At least one event is required")
		return
	}

	// Project scope enforcement: a key bound to a project may only report for
	// that project; a global key may report any project.
	keyProject, _ := c.Get("api_key_project")
	kp, _ := keyProject.(string)

	for _, req := range reqs {
		if err := validator.Validate(req); err != nil {
			res.BadRequest(c, validator.ValidationError(err))
			return
		}
		if kp != "" && req.Project != "" && req.Project != kp {
			res.Forbidden(c, fmt.Sprintf("API key is scoped to project %q", kp))
			return
		}
		if req.Project == "" {
			req.Project = kp
		}
	}

	results, err := h.controller.IngestBatch(c.Request.Context(), reqs)
	if err != nil {
		res.BadRequest(c, err.Error())
		return
	}

	res.Created(c, "events received successfully", map[string]interface{}{
		"received": len(results),
		"events":   results,
	})
}

// bindEvents parses either a single object or an array into a slice of requests.
func bindEvents(c *gin.Context) ([]*model.IngestEventRequest, error) {
	raw, err := c.GetRawData()
	if err != nil {
		return nil, fmt.Errorf("Invalid request body")
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("Request body is required")
	}

	if trimmed[0] == '[' {
		var reqs []*model.IngestEventRequest
		if err := json.Unmarshal(trimmed, &reqs); err != nil {
			return nil, fmt.Errorf("Invalid request body")
		}
		return reqs, nil
	}

	var req model.IngestEventRequest
	if err := json.Unmarshal(trimmed, &req); err != nil {
		return nil, fmt.Errorf("Invalid request body")
	}
	return []*model.IngestEventRequest{&req}, nil
}
