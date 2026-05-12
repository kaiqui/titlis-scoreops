package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/titlis/scoreops/internal/notifier"
	"github.com/titlis/scoreops/internal/scoring"
)

type snapshotStore interface {
	UpsertSnapshot(ctx context.Context, snap scoring.WorkloadSnapshot, rulesHash string) error
}

type scoreStore interface {
	UpsertScore(ctx context.Context, result scoring.ScoreResult) error
	AppendHistory(ctx context.Context, result scoring.ScoreResult, triggerType string) error
}

type EvaluateHandler struct {
	engine       *scoring.ScoreEngine
	resolver     *scoring.ContextResolver
	snapshots    snapshotStore
	scores       scoreStore
	notif        notifier.ScorecardNotifier
}

func NewEvaluateHandler(
	engine *scoring.ScoreEngine,
	resolver *scoring.ContextResolver,
	snapshots snapshotStore,
	scores scoreStore,
	notif notifier.ScorecardNotifier,
) *EvaluateHandler {
	return &EvaluateHandler{
		engine:    engine,
		resolver:  resolver,
		snapshots: snapshots,
		scores:    scores,
		notif:     notif,
	}
}

func (h *EvaluateHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var snap scoring.WorkloadSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateSnapshot(snap); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	activeRules, hash, err := h.resolver.ResolveActiveRules(r.Context(), snap.TenantID, snap.EngineSlug, snap)
	if err != nil {
		slog.Error("failed to resolve active rules", "err", err, "uid", snap.UID, "tenant", snap.TenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	weights, err := h.resolver.ResolveWeights(r.Context(), snap.TenantID, snap.EngineSlug)
	if err != nil {
		slog.Error("failed to resolve weights", "err", err, "uid", snap.UID, "tenant", snap.TenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	result := h.engine.Evaluate(snap, activeRules, weights)
	result.RulesHash = hash

	slog.Info("evaluate: completed",
		"uid", snap.UID,
		"workload", snap.Name,
		"namespace", snap.Namespace,
		"tenant", snap.TenantID,
		"score", result.OverallScore,
		"compliance", result.ComplianceStatus,
		"findings", len(result.Findings),
	)

	if err := h.snapshots.UpsertSnapshot(r.Context(), snap, hash); err != nil {
		slog.Error("failed to upsert snapshot", "err", err, "uid", snap.UID, "tenant", snap.TenantID)
	}

	if err := h.scores.UpsertScore(r.Context(), result); err != nil {
		slog.Error("failed to upsert score", "err", err, "uid", snap.UID, "tenant", snap.TenantID)
	} else if err := h.scores.AppendHistory(r.Context(), result, "operator_event"); err != nil {
		slog.Error("failed to append score history", "err", err, "uid", snap.UID, "tenant", snap.TenantID)
	}

	go h.notif.SendScorecardEvaluated(context.Background(), result)

	respondJSON(w, http.StatusOK, result)
}

func validateSnapshot(snap scoring.WorkloadSnapshot) error {
	switch {
	case snap.UID == "":
		return fmt.Errorf("uid é obrigatório")
	case snap.Name == "":
		return fmt.Errorf("name é obrigatório")
	case snap.Namespace == "":
		return fmt.Errorf("namespace é obrigatório")
	case snap.Cluster == "":
		return fmt.Errorf("cluster é obrigatório")
	case snap.TenantID == 0:
		return fmt.Errorf("tenant_id é obrigatório")
	case snap.EngineSlug == "":
		return fmt.Errorf("engine_slug é obrigatório")
	}
	return nil
}
