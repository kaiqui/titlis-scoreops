package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/repository"
	"github.com/titlis/scoreops/internal/worker"
)

type overrideStore interface {
	List(ctx context.Context, tenantID int, engineSlug, scope, cluster string) ([]domain.Override, error)
	Upsert(ctx context.Context, tenantID int, req domain.CreateOverrideRequest) (*domain.Override, error)
	GetByID(ctx context.Context, tenantID int, id int64) (*domain.Override, error)
	Delete(ctx context.Context, tenantID int, id int64) error
	Resolve(ctx context.Context, tenantID int, q domain.ResolveQuery) (*domain.ResolveResult, error)
}

type engineSlugResolver interface {
	SlugForID(ctx context.Context, engineID int) (string, error)
}

type uidsByScope interface {
	ListUIDsInScope(ctx context.Context, tenantID int64, engineSlug, scope, cluster, namespace, workloadUID string) ([]string, error)
}

type jobEnqueuer interface {
	Enqueue(job worker.RecalcJob)
}

type OverrideHandler struct {
	overrides overrideStore
	audit     *repository.AuditRepo
	engines   engineSlugResolver
	snapshots uidsByScope
	recalc    jobEnqueuer
}

func NewOverrideHandler(
	overrides overrideStore,
	audit *repository.AuditRepo,
	engines engineSlugResolver,
	snapshots uidsByScope,
	recalc jobEnqueuer,
) *OverrideHandler {
	return &OverrideHandler{
		overrides: overrides,
		audit:     audit,
		engines:   engines,
		snapshots: snapshots,
		recalc:    recalc,
	}
}

func (h *OverrideHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	q := r.URL.Query()
	overrides, err := h.overrides.List(r.Context(), tenantID,
		q.Get("engine"), q.Get("scope"), q.Get("cluster"))
	if err != nil {
		slog.Error("overrides: list failed", "err", err, "tenant", tenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if overrides == nil {
		overrides = []domain.Override{}
	}
	respondJSON(w, http.StatusOK, overrides)
}

func (h *OverrideHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	var req domain.CreateOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := req.Validate(); err != nil {
		handleDomainError(w, err)
		return
	}
	override, err := h.overrides.Upsert(r.Context(), tenantID, req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	h.audit.Append(r.Context(), tenantID, req.CreatedBy, "override_upserted",
		"override", strconv.FormatInt(override.ID, 10), nil, override)

	h.enqueueRuleChange(r.Context(), tenantID, req.EngineID,
		string(req.Scope), req.ClusterName, req.Namespace, req.WorkloadUID)

	respondJSON(w, http.StatusCreated, override)
}

func (h *OverrideHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	// Fetch scope before deleting so we know which UIDs are affected.
	existing, err := h.overrides.GetByID(r.Context(), tenantID, id)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	if err := h.overrides.Delete(r.Context(), tenantID, id); err != nil {
		handleDomainError(w, err)
		return
	}
	h.audit.Append(r.Context(), tenantID, "system", "override_deleted",
		"override", strconv.FormatInt(id, 10), map[string]int64{"id": id}, nil)

	h.enqueueRuleChange(r.Context(), tenantID, existing.EngineID,
		string(existing.Scope), existing.ClusterName, existing.Namespace, existing.WorkloadUID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *OverrideHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	q := r.URL.Query()
	result, err := h.overrides.Resolve(r.Context(), tenantID, domain.ResolveQuery{
		Engine:      q.Get("engine"),
		RuleID:      q.Get("ruleId"),
		ClusterName: q.Get("cluster"),
		Namespace:   q.Get("namespace"),
		WorkloadUID: q.Get("workloadUid"),
	})
	if err != nil {
		slog.Error("overrides: resolve failed", "err", err, "tenant", tenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func tenantIDFromPath(r *http.Request) int {
	n, _ := strconv.Atoi(chi.URLParam(r, "tenantId"))
	return n
}

func (h *OverrideHandler) enqueueRuleChange(ctx context.Context, tenantID, engineID int, scope, cluster, namespace, workloadUID string) {
	engineSlug, err := h.engines.SlugForID(ctx, engineID)
	if err != nil {
		slog.Warn("recalc trigger: could not resolve engine slug", "engine_id", engineID, "err", err)
		return
	}

	uids, err := h.snapshots.ListUIDsInScope(ctx, int64(tenantID), engineSlug, scope, cluster, namespace, workloadUID)
	if err != nil {
		slog.Warn("recalc trigger: could not list uids", "engine_slug", engineSlug, "err", err)
		return
	}

	h.recalc.Enqueue(worker.RecalcJob{
		TenantID:    int64(tenantID),
		EngineSlug:  engineSlug,
		UIDs:        uids,
		TriggerType: "rule_change",
	})
}
