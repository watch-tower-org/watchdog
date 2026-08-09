package notifier

import (
	"context"
	"sync"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/cache"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

const (
	defaultWorkers     = 5
	defaultQueueSize   = 1000
	defaultEvalTimeout = 30 * time.Second
)

type job struct {
	IssueID       int64
	IsNewIssue    bool
	WasRegression bool
	Project       string
	Tag           string
}

// Caches holds the shared in-memory caches used by the notification worker.
// The same instances are owned by the settings/rules/recipient controllers,
// which invalidate them on writes; the worker only reads from them.
type Caches struct {
	AlertSettings *cache.SingletonCache[model.AlertSettings]
	EmailSettings *cache.SingletonCache[model.EmailSettings]
	Rules         *cache.MapCache[int64, model.AlertRule]
	Recipients    *cache.MapCache[int64, model.RecipientList]
}

// Notifier evaluates alert rules for ingested events and sends throttled
// email notifications. Evaluation runs on a worker pool so ingestion is never
// blocked by SMTP or rule evaluation.
type Notifier struct {
	db     *bun.DB
	caches *Caches
	jobs   chan job
	stop   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewNotifier(db *bun.DB, workers int, caches *Caches) *Notifier {
	if workers < 1 {
		workers = defaultWorkers
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Notifier{
		db:     db,
		caches: caches,
		jobs:   make(chan job, defaultQueueSize),
		stop:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start launches the worker pool. Safe to call once.
func (n *Notifier) Start() {
	for i := 0; i < defaultWorkers; i++ {
		n.wg.Add(1)
		go n.worker()
	}
	logger.Info().Msgf("notification worker pool started (%d workers)", defaultWorkers)
}

// Shutdown stops the worker pool and waits for in-flight jobs with a timeout.
func (n *Notifier) Shutdown() {
	n.cancel()
	close(n.stop)
	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		logger.Warn().Msg("notification worker shutdown timed out")
	}
}

// NotifyAsync enqueues an evaluation job. Never blocks: if the queue is full
// the event is dropped (logged) rather than stalling ingestion.
func (n *Notifier) NotifyAsync(issueID int64, isNewIssue, wasRegression bool, project, tag string) {
	select {
	case n.jobs <- job{
		IssueID:       issueID,
		IsNewIssue:    isNewIssue,
		WasRegression: wasRegression,
		Project:       project,
		Tag:           tag,
	}:
	default:
		logger.Warn().Msgf("notification queue full, dropped evaluation for issue %d", issueID)
	}
}

func (n *Notifier) worker() {
	defer n.wg.Done()
	for {
		select {
		case <-n.stop:
			return
		case j, ok := <-n.jobs:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(n.ctx, defaultEvalTimeout)
			if err := n.evaluate(ctx, j); err != nil {
				logger.Ctx(ctx).Error().Msgf("notification evaluation failed for issue %d: %v", j.IssueID, err)
			}
			cancel()
		}
	}
}
