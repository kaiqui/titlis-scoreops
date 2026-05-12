package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/titlis/scoreops/internal/repository"
	"github.com/titlis/scoreops/internal/scoring"
)

type scoreReader interface {
	ListScores(ctx context.Context, tenantID int64, engineSlug, cluster, namespace string, limit int) ([]scoring.ScoreResult, error)
	GetScore(ctx context.Context, tenantID int64, engineSlug, workloadUID string) (*scoring.ScoreResult, []repository.ScoreHistoryEntry, error)
}

type ScoresHandler struct {
	scores scoreReader
}

func NewScoresHandler(scores scoreReader) *ScoresHandler {
	return &ScoresHandler{scores: scores}
}

type scoreWithHistory struct {
	Score   scoring.ScoreResult           `json:"score"`
	History []repository.ScoreHistoryEntry `json:"history"`
}

func (h *ScoresHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(tenantIDFromPath(r))
	q := r.URL.Query()
	engine := q.Get("engine")
	if engine == "" {
		engine = "kubernetes"
	}
	limit, _ := strconv.Atoi(q.Get("limit"))

	results, err := h.scores.ListScores(r.Context(), tenantID, engine,
		q.Get("cluster"), q.Get("namespace"), limit)
	if err != nil {
		slog.Error("scores: list failed", "err", err, "tenant", tenantID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if results == nil {
		results = []scoring.ScoreResult{}
	}
	respondJSON(w, http.StatusOK, results)
}

func (h *ScoresHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := int64(tenantIDFromPath(r))
	workloadUID := chi.URLParam(r, "workloadUid")
	engine := r.URL.Query().Get("engine")
	if engine == "" {
		engine = "kubernetes"
	}

	result, history, err := h.scores.GetScore(r.Context(), tenantID, engine, workloadUID)
	if err != nil {
		slog.Error("scores: get failed", "err", err, "tenant", tenantID, "uid", workloadUID)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if result == nil {
		respondError(w, http.StatusNotFound, "score não encontrado")
		return
	}
	if history == nil {
		history = []repository.ScoreHistoryEntry{}
	}
	respondJSON(w, http.StatusOK, scoreWithHistory{Score: *result, History: history})
}
