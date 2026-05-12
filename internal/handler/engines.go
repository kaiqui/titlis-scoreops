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
)

type engineStore interface {
	List(ctx context.Context) ([]domain.Engine, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Engine, error)
	Create(ctx context.Context, req domain.CreateEngineRequest) (*domain.Engine, error)
	SetEnabled(ctx context.Context, slug string, enabled bool) error
	ListRules(ctx context.Context, engineID int) ([]domain.Rule, error)
	CreateRule(ctx context.Context, engineID int, req domain.CreateRuleRequest) (*domain.Rule, error)
}

type EngineHandler struct {
	engines engineStore
	audit   *repository.AuditRepo
}

func NewEngineHandler(engines engineStore, audit *repository.AuditRepo) *EngineHandler {
	return &EngineHandler{engines: engines, audit: audit}
}

func (h *EngineHandler) List(w http.ResponseWriter, r *http.Request) {
	engines, err := h.engines.List(r.Context())
	if err != nil {
		slog.Error("engines: list failed", "err", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if engines == nil {
		engines = []domain.Engine{}
	}
	respondJSON(w, http.StatusOK, engines)
}

func (h *EngineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateEngineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := req.Validate(); err != nil {
		handleDomainError(w, err)
		return
	}
	engine, err := h.engines.Create(r.Context(), req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	h.audit.Append(r.Context(), 0, "system", "engine_created", "engine", engine.Slug, nil, engine)
	respondJSON(w, http.StatusCreated, engine)
}

func (h *EngineHandler) PatchEnabled(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req domain.PatchEngineEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := h.engines.SetEnabled(r.Context(), slug, req.Enabled); err != nil {
		handleDomainError(w, err)
		return
	}
	h.audit.Append(r.Context(), 0, "system", "engine_toggled", "engine", slug, nil, req)
	respondJSON(w, http.StatusOK, map[string]any{"slug": slug, "enabled": req.Enabled})
}

func (h *EngineHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	engine, err := h.engines.GetBySlug(r.Context(), slug)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	rules, err := h.engines.ListRules(r.Context(), engine.ID)
	if err != nil {
		slog.Error("engines: list rules failed", "err", err, "engine", slug)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if rules == nil {
		rules = []domain.Rule{}
	}
	respondJSON(w, http.StatusOK, rules)
}

func (h *EngineHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	engine, err := h.engines.GetBySlug(r.Context(), slug)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	var req domain.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := req.Validate(); err != nil {
		handleDomainError(w, err)
		return
	}
	rule, err := h.engines.CreateRule(r.Context(), engine.ID, req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	h.audit.Append(r.Context(), 0, "system", "rule_created", "rule",
		slug+"/"+rule.RuleID, nil, rule)
	respondJSON(w, http.StatusCreated, rule)
}

func engineIDFromQuery(r *http.Request) int {
	s := r.URL.Query().Get("engine_id")
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}
