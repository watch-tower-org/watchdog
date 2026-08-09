// Package selfreport implements WatchTower dogfooding: the backend reports its
// own errors back into itself through the same ingestion/dedup/alert pipeline
// it exposes to other services.
//
// Unlike external SDKs, self-reports are delivered in-process: the SDK client
// is given a Sender that calls the ingestion controller directly, so no API
// key, HTTP round-trip, or configuration is required. This keeps the SDK as
// the capture/batching layer (panic-stack trimming, context enrichment) while
// the full fingerprinting, dedup, regression and alerting logic still runs.
//
// If the backend's own error is caused by Postgres being unavailable, the
// in-process ingest fails and is logged; a reentrancy guard prevents the
// failure from re-triggering self-reports in a loop. The regular stdout/file
// log output keeps working, so visibility is never lost exactly when things
// break hardest.
package selfreport

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/rs/zerolog"
	wt "github.com/watch-tower-org/watchtower/sdk/go"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

// ingester is the subset of the ingestion controller used to store
// self-reports. It lets tests substitute a stub.
type ingester interface {
	IngestBatch(ctx context.Context, reqs []*model.IngestEventRequest) ([]*model.IngestResult, error)
}

// sink delivers captured events to the in-process ingestion controller.
type sink struct {
	ingest ingester
	// inSelf is set while the ingestion controller is running so the log hook
	// can suppress re-reporting of failures produced by the ingest itself.
	inSelf atomic.Bool
}

func (s *sink) Send(ctx context.Context, events []wt.Event) ([]wt.Result, error) {
	if len(events) == 0 {
		return nil, nil
	}
	reqs := make([]*model.IngestEventRequest, 0, len(events))
	for i := range events {
		e := &events[i]
		reqs = append(reqs, &model.IngestEventRequest{
			Message:    e.Message,
			ErrorType:  e.ErrorType,
			StackTrace: e.StackTrace,
			Project:    e.Project,
			Tag:        e.Tag,
			Context:    e.Context,
			Timestamp:  e.Timestamp,
		})
	}

	s.inSelf.Store(true)
	defer s.inSelf.Store(false)

	results, err := s.ingest.IngestBatch(ctx, reqs)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("self-report: ingest failed: %v", err)
		return nil, err
	}

	out := make([]wt.Result, 0, len(results))
	for _, r := range results {
		out = append(out, wt.Result{
			IssueID:       r.IssueID,
			EventID:       r.EventID,
			IsNewIssue:    r.IsNewIssue,
			WasRegression: r.WasRegression,
			Fingerprint:   r.Fingerprint,
		})
	}
	return out, nil
}

// Reporter reports the backend's own errors to itself. A nil *Reporter (or a
// nil client) is a safe no-op.
type Reporter struct {
	client *wt.Client
	sink   *sink
}

// New builds a Reporter. With self-reporting disabled it returns (nil, nil).
// Any client setup failure returns an error so the caller can log it and
// continue without self-reporting.
func New(cfg config.SelfReportConfig, ingest ingester) (*Reporter, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	s := &sink{ingest: ingest}
	wcfg := wt.DefaultConfig()
	wcfg.Project = cfg.Project
	wcfg.Release = cfg.Release
	wcfg.Tag = "self"
	wcfg.Sender = s

	client, err := wt.NewClient(wcfg)
	if err != nil {
		return nil, fmt.Errorf("self-report: init sdk client: %w", err)
	}

	r := &Reporter{client: client, sink: s}
	r.installLogHook(cfg.Level)
	logger.Info().Msgf("self-reporting enabled: project=%s", cfg.Project)
	return r, nil
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
		// Failures produced while running the ingest itself must not spawn a
		// second generation of self-reports (e.g. during a DB outage).
		if r.sink != nil && r.sink.inSelf.Load() {
			return
		}
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
