package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/titlis/scoreops/internal/coverage"
)

// CoverageEvaluateHandler computes a personalized coverage scorecard for one service.
// v1: compute + return (persistence/notification = D5e). Findings are deterministic — no AI.
type CoverageEvaluateHandler struct {
	engine *coverage.Engine
}

func NewCoverageEvaluateHandler(engine *coverage.Engine) *CoverageEvaluateHandler {
	return &CoverageEvaluateHandler{engine: engine}
}

func (h *CoverageEvaluateHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var snap coverage.CoverageSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateCoverageSnapshot(snap); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	result := h.engine.Evaluate(snap)

	slog.Info("coverage: completed",
		"uid", snap.WorkloadUID,
		"service", snap.ServiceName,
		"tenant", snap.TenantID,
		"trust", result.TrustScore,
		"findings", len(result.Findings),
	)

	respondJSON(w, http.StatusOK, result)
}

func validateCoverageSnapshot(snap coverage.CoverageSnapshot) error {
	switch {
	case snap.WorkloadUID == "":
		return fmt.Errorf("workloadUid é obrigatório")
	case snap.TenantID == 0:
		return fmt.Errorf("tenantId é obrigatório")
	}
	return nil
}
