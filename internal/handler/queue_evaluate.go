package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

type queueSnapshotStore interface {
	Upsert(ctx context.Context, snap domain.QueueSnapshot) error
}

type queueScoreStore interface {
	Upsert(ctx context.Context, result scoring.QueueScoreResult) error
}

type queueNotifier interface {
	SendQueueEvaluated(ctx context.Context, result scoring.QueueScoreResult)
}

// labelEntry is the wire format for label registry entries from titlis-api: [{key, values}].
type labelEntry struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type QueueEvaluateRequest struct {
	Provider    string `json:"provider"`
	ExternalID  string `json:"externalId"`
	DisplayName string `json:"displayName"`
	ProjectID   string `json:"projectId"`
	TopicID     string `json:"topicId"`
	IsDLQ       bool   `json:"isDlq"`
	TenantID    int64  `json:"tenantId"`

	NumUndeliveredMessages      int64   `json:"numUndeliveredMessages"`
	OldestUnackedAgeSec         int64   `json:"oldestUnackedAgeSec"`
	PullMessageCountRate        float64 `json:"pullMessageCountRate"`
	SendMessageCountRate        float64 `json:"sendMessageCountRate"`
	AckMessageCountRate         float64 `json:"ackMessageCountRate"`
	DeadLetterMessageCount      int64   `json:"deadLetterMessageCount"`
	MessageRetentionDurationSec int64   `json:"messageRetentionDurationSec"`

	HasDLQConfigured  bool              `json:"hasDlqConfigured"`
	HasSnapshotPolicy bool              `json:"hasSnapshotPolicy"`
	Labels            map[string]string `json:"labels"`

	HasMonitorBacklog bool `json:"hasMonitorBacklog"`
	HasMonitorAge     bool `json:"hasMonitorAge"`
	HasMonitorDLQ     bool `json:"hasMonitorDlq"`

	Thresholds    domain.QueueThresholds `json:"thresholds"`
	LabelRegistry []labelEntry           `json:"labelRegistry"`

	CollectedAt time.Time `json:"collectedAt"`
}

func (r QueueEvaluateRequest) labelRegistry() domain.LabelRegistry {
	reg := make(domain.LabelRegistry, len(r.LabelRegistry))
	for _, e := range r.LabelRegistry {
		reg[e.Key] = e.Values
	}
	return reg
}

func (r QueueEvaluateRequest) toSnapshot() domain.QueueSnapshot {
	return domain.QueueSnapshot{
		Provider:                    r.Provider,
		ExternalID:                  r.ExternalID,
		DisplayName:                 r.DisplayName,
		ProjectID:                   r.ProjectID,
		TopicID:                     r.TopicID,
		IsDLQ:                       r.IsDLQ,
		TenantID:                    r.TenantID,
		NumUndeliveredMessages:      r.NumUndeliveredMessages,
		OldestUnackedAgeSec:         r.OldestUnackedAgeSec,
		PullMessageCountRate:        r.PullMessageCountRate,
		SendMessageCountRate:        r.SendMessageCountRate,
		AckMessageCountRate:         r.AckMessageCountRate,
		DeadLetterMessageCount:      r.DeadLetterMessageCount,
		MessageRetentionDurationSec: r.MessageRetentionDurationSec,
		HasDLQConfigured:            r.HasDLQConfigured,
		HasSnapshotPolicy:           r.HasSnapshotPolicy,
		Labels:                      r.Labels,
		HasMonitorBacklog:           r.HasMonitorBacklog,
		HasMonitorAge:               r.HasMonitorAge,
		HasMonitorDLQ:               r.HasMonitorDLQ,
		CollectedAt:                 r.CollectedAt,
	}
}

type QueueEvaluateHandler struct {
	engine    *scoring.QueueScoreEngine
	snapshots queueSnapshotStore
	scores    queueScoreStore
	notif     queueNotifier
}

func NewQueueEvaluateHandler(
	engine    *scoring.QueueScoreEngine,
	snapshots queueSnapshotStore,
	scores    queueScoreStore,
	notif     queueNotifier,
) *QueueEvaluateHandler {
	return &QueueEvaluateHandler{engine: engine, snapshots: snapshots, scores: scores, notif: notif}
}

func (h *QueueEvaluateHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req QueueEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateQueueRequest(req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	snap := req.toSnapshot()
	active := h.engine.AllRulesActive()
	result := h.engine.Evaluate(snap, active, req.Thresholds, req.labelRegistry(), nil)

	slog.Info("queue evaluate: completed",
		"external_id", snap.ExternalID,
		"provider", snap.Provider,
		"tenant", snap.TenantID,
		"score", result.OverallScore,
		"compliance", result.ComplianceStatus,
		"findings", len(result.Findings),
	)

	if err := h.snapshots.Upsert(r.Context(), snap); err != nil {
		slog.Error("queue: failed to upsert snapshot", "err", err, "external_id", snap.ExternalID)
	}

	if err := h.scores.Upsert(r.Context(), result); err != nil {
		slog.Error("queue: failed to upsert score", "err", err, "external_id", snap.ExternalID)
	}

	go h.notif.SendQueueEvaluated(context.Background(), result)

	respondJSON(w, http.StatusOK, result)
}

func validateQueueRequest(req QueueEvaluateRequest) error {
	switch {
	case req.ExternalID == "":
		return fmt.Errorf("external_id é obrigatório")
	case req.Provider == "":
		return fmt.Errorf("provider é obrigatório")
	case req.TenantID == 0:
		return fmt.Errorf("tenant_id é obrigatório")
	}
	return nil
}
