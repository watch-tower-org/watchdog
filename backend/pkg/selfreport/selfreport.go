// Package selfreport implements WatchTower dogfooding: the backend reports its
// own errors back into itself through the same SDK/ingestion pipeline it
// exposes to other services.
//
// If the backend's own error is caused by Postgres being unavailable, an
// HTTP self-report cannot be persisted. In that case the SDK drops the event
// and logs to stderr, and the existing zerolog file/stdout output still works,
// so visibility is never lost while things break hardest.
package selfreport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	wt "github.com/watch-tower-org/watchdog/sdk/go"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

// keyFileName is the default file holding the plaintext self-reporting API key.
const keyFileName = "self-report.key"

// apiKeyCreator is the subset of the api_keys controller used to provision the
// self-reporting key.
type apiKeyCreator interface {
	Create(ctx context.Context, req *model.CreateApiKeyRequest) (*model.CreateApiKeyResponse, error)
}

// Reporter reports the backend's own errors to itself. A nil *Reporter (or a
// nil client) is a safe no-op.
type Reporter struct {
	client *wt.Client
}

// New builds a Reporter from cfg. With self-reporting disabled it returns
// (nil, nil). Any provisioning or client setup failure returns an error so the
// caller can log it and continue without self-reporting.
func New(cfg config.SelfReportConfig, logDir string, keys apiKeyCreator) (*Reporter, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	keyFile := cfg.KeyFile
	if keyFile == "" {
		keyFile = filepath.Join(logDir, keyFileName)
	}

	key, fromFile, err := resolveKey(cfg.APIKey, keyFile, os.ReadFile, func() (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		res, err := keys.Create(ctx, &model.CreateApiKeyRequest{
			Name:    cfg.Project,
			Project: cfg.Project,
		})
		if err != nil {
			return "", err
		}
		return res.Key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("self-report: provision api key: %w", err)
	}
	if !fromFile && cfg.APIKey == "" {
		if err := writeKeyFile(keyFile, key); err != nil {
			logger.Warn().Err(err).Msg("self-report: could not persist api key; a new one will be created next boot")
		}
	}

	wcfg := wt.DefaultConfig()
	wcfg.BaseURL = cfg.BaseURL
	wcfg.APIKey = key
	wcfg.Project = cfg.Project
	wcfg.Release = cfg.Release
	wcfg.Tag = "self"

	client, err := wt.NewClient(wcfg)
	if err != nil {
		return nil, fmt.Errorf("self-report: init sdk client: %w", err)
	}

	r := &Reporter{client: client}
	r.installLogHook(cfg.Level)
	logger.Info().Msgf("self-reporting enabled: project=%s base=%s", cfg.Project, cfg.BaseURL)
	return r, nil
}

// resolveKey returns the self-reporting API key: an explicit override wins,
// then a previously persisted key file, then a freshly provisioned key.
// fromFile reports whether the returned key came from the key file.
func resolveKey(cfgKey, keyFile string, readFile func(string) ([]byte, error), provision func() (string, error)) (key string, fromFile bool, err error) {
	if strings.TrimSpace(cfgKey) != "" {
		return strings.TrimSpace(cfgKey), false, nil
	}
	if keyFile != "" {
		if data, readErr := readFile(keyFile); readErr == nil {
			if k := strings.TrimSpace(string(data)); k != "" {
				return k, true, nil
			}
		}
	}
	k, err := provision()
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(k), false, nil
}

func writeKeyFile(path, key string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(key+"\n"), 0o600)
}

// Report enqueues err for delivery. Safe to call when self-reporting is off.
func (r *Reporter) Report(err error, opts ...wt.ReportOption) {
	if r == nil || r.client == nil {
		return
	}
	r.client.Report(err, opts...)
}

// ReportPanic reports a recovered panic value with the panic-time stack.
func (r *Reporter) ReportPanic(v any, opts ...wt.ReportOption) {
	if r == nil || r.client == nil {
		return
	}
	r.client.ReportPanic(v, opts...)
}

// Recover reports a recovered panic with request context. It is the callback
// used by the gin recovery middleware.
func (r *Reporter) Recover(v any, ctx map[string]any) {
	if r == nil || r.client == nil {
		return
	}
	opts := []wt.ReportOption{wt.WithTag("panic.recovered")}
	if len(ctx) > 0 {
		opts = append(opts, wt.WithContext(ctx))
	}
	r.client.ReportPanic(v, opts...)
}

// Close flushes buffered self-reports and stops the client. Safe to call
// multiple times.
func (r *Reporter) Close() {
	if r != nil && r.client != nil {
		r.client.Close()
	}
}

// installLogHook forwards zerolog messages at or above cfgLevel to the SDK.
// Fatal/Panic are reported synchronously because zerolog exits the process
// immediately after writing them; lower levels are batched asynchronously.
func (r *Reporter) installLogHook(cfgLevel string) {
	min := logger.ParseLevel(cfgLevel)
	if min == zerolog.Disabled {
		return
	}
	logger.InstallLevelHook(min, func(level zerolog.Level, message string) {
		opts := []wt.ReportOption{wt.WithTag("log." + strings.ToLower(level.String()))}
		err := fmt.Errorf("%s", message)
		if level >= zerolog.FatalLevel {
			if _, serr := r.client.ReportSync(err, opts...); serr != nil {
				logger.Warn().Err(serr).Msg("self-report: failed to report fatal log")
			}
			return
		}
		r.client.Report(err, opts...)
	})
}
