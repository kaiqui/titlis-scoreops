package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/titlis/scoreops/internal/repository"
)

type AuditHandler struct{ audit *repository.AuditRepo }

func NewAuditHandler(audit *repository.AuditRepo) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromPath(r)
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)

	entries, err := h.audit.List(r.Context(), tenantID, limit, before)
	if err != nil {
		slog.Error("audit: list failed", "err", err, "tenant", tenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if entries == nil {
		entries = []repository.AuditEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}
