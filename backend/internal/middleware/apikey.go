package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/res"
)

const (
	APIKeyHeader = "X-Api-Key"
	// CtxApiKeyID is the gin context key holding the authenticated api key id.
	CtxApiKeyID = "api_key_id"
	// CtxApiKeyProject is the gin context key holding the key's project (may be empty for global keys).
	CtxApiKeyProject = "api_key_project"
)

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// ApiKeyAuth authenticates SDK requests using an API key. The key is provided
// via the X-Api-Key header (Authorization: Bearer is also accepted). The key's
// stored SHA-256 hash is looked up; revoked or unknown keys are rejected.
func ApiKeyAuth(db *bun.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractApiKey(c)
		if key == "" {
			res.Unauthorized(c, "API key is required")
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var ak model.ApiKey
		err := db.NewSelect().
			Model(&ak).
			Where("key_hash = ?", hashKey(key)).
			Limit(1).
			Scan(ctx)

		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				logger.Ctx(ctx).Error().Msgf("api key lookup failed: %v", err)
			}
			res.Unauthorized(c, "Invalid API key")
			c.Abort()
			return
		}

		if !ak.IsActive {
			res.Unauthorized(c, "API key has been revoked")
			c.Abort()
			return
		}

		c.Set(CtxApiKeyID, ak.ID)
		c.Set(CtxApiKeyProject, ak.Project)
		c.Next()

		// Best-effort: refresh last_used_at in the background so the hot path
		// isn't blocked by a write on every event.
		go func(id int64) {
			bctx, bcancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer bcancel()
			_, err := db.NewUpdate().
				Model((*model.ApiKey)(nil)).
				Set("last_used_at = ?", time.Now()).
				Where("id = ?", id).
				Exec(bctx)
			if err != nil {
				logger.Error().Msgf("failed to update api key last_used_at id=%d: %v", id, err)
			}
		}(ak.ID)
	}
}

func extractApiKey(c *gin.Context) string {
	if key := c.GetHeader(APIKeyHeader); key != "" {
		return strings.TrimSpace(key)
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}
	return ""
}
