package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/titlis/scoreops/internal/domain"
)

type tagPolicyStore interface {
	List(ctx context.Context, tenantID int64) ([]domain.TagPolicy, error)
	Create(ctx context.Context, tenantID int64, createdBy string, req domain.CreateTagPolicyRequest) (*domain.TagPolicy, error)
	Delete(ctx context.Context, tenantID, id int64) error
}

type TagPolicyHandler struct {
	policies tagPolicyStore
}

func NewTagPolicyHandler(policies tagPolicyStore) *TagPolicyHandler {
	return &TagPolicyHandler{policies: policies}
}

func (h *TagPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(tenantIDFromPath(r))
	policies, err := h.policies.List(r.Context(), tenantID)
	if err != nil {
		slog.Error("tag-policies: list failed", "err", err, "tenant", tenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if policies == nil {
		policies = []domain.TagPolicy{}
	}
	respondJSON(w, http.StatusOK, policies)
}

func (h *TagPolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(tenantIDFromPath(r))
	var req domain.CreateTagPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := req.Validate(); err != nil {
		handleDomainError(w, err)
		return
	}
	policy, err := h.policies.Create(r.Context(), tenantID, req.CreatedBy, req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, policy)
}

func (h *TagPolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(tenantIDFromPath(r))
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.policies.Delete(r.Context(), tenantID, id); err != nil {
		handleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
