package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/repository"
	"github.com/titlis/scoreops/internal/worker"
)

type weightStore interface {
	Get(ctx context.Context, tenantID, engineID int) ([]domain.PillarWeight, error)
	Set(ctx context.Context, tenantID int, req domain.SetWeightsRequest) ([]domain.PillarWeight, error)
}

type allUIDsLister interface {
	ListAllUIDs(ctx context.Context, tenantID int64, engineSlug string) ([]string, error)
}

type WeightHandler struct {
	weights   weightStore
	audit     *repository.AuditRepo
	engines   engineSlugResolver
	snapshots allUIDsLister
	recalc    jobEnqueuer
	minWeight int
	maxWeight int
}

func NewWeightHandler(
	weights weightStore,
	audit *repository.AuditRepo,
	engines engineSlugResolver,
	snapshots allUIDsLister,
	recalc jobEnqueuer,
	min, max int,
) *WeightHandler {
	return &WeightHandler{
		weights:   weights,
		audit:     audit,
		engines:   engines,
		snapshots: snapshots,
		recalc:    recalc,
		minWeight: min,
		maxWeight: max,
	}
}

func (h *WeightHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	engineIDStr := r.URL.Query().Get("engine_id")
	engineID, _ := strconv.Atoi(engineIDStr)

	weights, err := h.weights.Get(r.Context(), tenantID, engineID)
	if err != nil {
		slog.Error("weights: get failed", "err", err, "tenant", tenantID, "engine_id", engineID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if weights == nil {
		weights = []domain.PillarWeight{}
	}
	respondJSON(w, http.StatusOK, weights)
}

func (h *WeightHandler) Set(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	var req domain.SetWeightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := req.Validate(h.minWeight, h.maxWeight); err != nil {
		handleDomainError(w, err)
		return
	}

	previous, _ := h.weights.Get(r.Context(), tenantID, req.EngineID)
	result, err := h.weights.Set(r.Context(), tenantID, req)
	if err != nil {
		slog.Error("weights: set failed", "err", err, "tenant", tenantID, "engine_id", req.EngineID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.audit.Append(r.Context(), tenantID, req.UpdatedBy, "weights_updated",
		"weights", strconv.Itoa(req.EngineID), previous, result)

	h.enqueueWeightChange(r.Context(), tenantID, req.EngineID)

	respondJSON(w, http.StatusOK, result)
}

func (h *WeightHandler) enqueueWeightChange(ctx context.Context, tenantID, engineID int) {
	engineSlug, err := h.engines.SlugForID(ctx, engineID)
	if err != nil {
		slog.Warn("recalc trigger: could not resolve engine slug", "engine_id", engineID, "err", err)
		return
	}

	uids, err := h.snapshots.ListAllUIDs(ctx, int64(tenantID), engineSlug)
	if err != nil {
		slog.Warn("recalc trigger: could not list uids", "engine_slug", engineSlug, "err", err)
		return
	}

	h.recalc.Enqueue(worker.RecalcJob{
		TenantID:    int64(tenantID),
		EngineSlug:  engineSlug,
		UIDs:        uids,
		TriggerType: "weight_change",
	})
}
