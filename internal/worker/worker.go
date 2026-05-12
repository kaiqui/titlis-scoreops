package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/titlis/scoreops/internal/notifier"
	"github.com/titlis/scoreops/internal/scoring"
)

// RecalcJob describes a recalculation request triggered by a config change.
type RecalcJob struct {
	TenantID    int64
	EngineSlug  string
	UIDs        []string // nil = all workloads for the tenant/engine
	TriggerType string   // "rule_change" | "weight_change"
}

type snapshotLoader interface {
	GetSnapshot(ctx context.Context, tenantID int64, engineSlug, workloadUID string) (*scoring.WorkloadSnapshot, error)
	ListAllUIDs(ctx context.Context, tenantID int64, engineSlug string) ([]string, error)
}

type scoreWriter interface {
	UpsertScore(ctx context.Context, result scoring.ScoreResult) error
	AppendHistory(ctx context.Context, result scoring.ScoreResult, triggerType string) error
}

// RecalcWorker processes recalculation jobs in the background.
// One goroutine drains the channel; each UID is retried up to 3 times with exponential backoff.
type RecalcWorker struct {
	jobs      chan RecalcJob
	snapshots snapshotLoader
	scores    scoreWriter
	engine    *scoring.ScoreEngine
	resolver  *scoring.ContextResolver
	notif     notifier.ScorecardNotifier
}

func NewRecalcWorker(
	bufSize int,
	snapshots snapshotLoader,
	scores scoreWriter,
	engine *scoring.ScoreEngine,
	resolver *scoring.ContextResolver,
	notif notifier.ScorecardNotifier,
) *RecalcWorker {
	return &RecalcWorker{
		jobs:      make(chan RecalcJob, bufSize),
		snapshots: snapshots,
		scores:    scores,
		engine:    engine,
		resolver:  resolver,
		notif:     notif,
	}
}

// Enqueue adds a job to the worker queue. Non-blocking: drops the job and logs a warning if the
// channel is full.
func (w *RecalcWorker) Enqueue(job RecalcJob) {
	select {
	case w.jobs <- job:
		slog.Info("recalc enqueued",
			"tenant", job.TenantID, "engine", job.EngineSlug,
			"uids", len(job.UIDs), "trigger", job.TriggerType)
	default:
		slog.Warn("recalc queue full — job dropped",
			"tenant", job.TenantID, "engine", job.EngineSlug, "trigger", job.TriggerType)
	}
}

// Start runs the processing loop. It returns when ctx is cancelled.
func (w *RecalcWorker) Start(ctx context.Context) {
	slog.Info("recalc worker started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("recalc worker stopping")
			return
		case job := <-w.jobs:
			w.processJob(ctx, job)
		}
	}
}

func (w *RecalcWorker) processJob(ctx context.Context, job RecalcJob) {
	uids := job.UIDs
	if len(uids) == 0 {
		var err error
		uids, err = w.snapshots.ListAllUIDs(ctx, job.TenantID, job.EngineSlug)
		if err != nil {
			slog.Error("recalc: list all uids failed", "err", err,
				"tenant", job.TenantID, "engine", job.EngineSlug)
			return
		}
	}

	slog.Info("recalc job started",
		"tenant", job.TenantID, "engine", job.EngineSlug,
		"total", len(uids), "trigger", job.TriggerType)

	succeeded, failed := 0, 0
	for _, uid := range uids {
		if err := w.recalcWithRetry(ctx, job, uid); err != nil {
			slog.Error("recalc: uid permanently failed",
				"uid", uid, "tenant", job.TenantID, "err", err)
			failed++
		} else {
			succeeded++
		}
	}

	slog.Info("recalc job finished",
		"tenant", job.TenantID, "engine", job.EngineSlug,
		"succeeded", succeeded, "failed", failed, "trigger", job.TriggerType)
}

func (w *RecalcWorker) recalcWithRetry(ctx context.Context, job RecalcJob, uid string) error {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := w.recalcUID(ctx, job, uid); err != nil {
			lastErr = err
			if attempt < maxAttempts {
				backoff := time.Duration(1<<(attempt-1)) * time.Second // 1s, 2s
				slog.Warn("recalc: retry",
					"uid", uid, "attempt", attempt, "backoff", backoff, "err", err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
			}
			continue
		}
		if attempt > 1 {
			slog.Info("recalc: retry succeeded", "uid", uid, "attempt", attempt)
		}
		return nil
	}
	return lastErr
}

func (w *RecalcWorker) recalcUID(ctx context.Context, job RecalcJob, uid string) error {
	snap, err := w.snapshots.GetSnapshot(ctx, job.TenantID, job.EngineSlug, uid)
	if err != nil {
		return err
	}
	if snap == nil {
		slog.Debug("recalc: no snapshot, skipping", "uid", uid)
		return nil
	}

	activeRules, hash, err := w.resolver.ResolveActiveRules(ctx, snap.TenantID, snap.EngineSlug, *snap)
	if err != nil {
		return err
	}
	weights, err := w.resolver.ResolveWeights(ctx, snap.TenantID, snap.EngineSlug)
	if err != nil {
		return err
	}

	result := w.engine.Evaluate(*snap, activeRules, weights)
	result.RulesHash = hash

	if err := w.scores.UpsertScore(ctx, result); err != nil {
		return err
	}
	if err := w.scores.AppendHistory(ctx, result, job.TriggerType); err != nil {
		return err
	}

	go w.notif.SendScorecardEvaluated(context.Background(), result)
	return nil
}
