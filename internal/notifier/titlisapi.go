package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/titlis/scoreops/internal/scoring"
)

type TitlisAPINotifier struct {
	baseURL        string
	internalSecret string
	client         *http.Client
}

func NewTitlisAPINotifier(baseURL, internalSecret string) *TitlisAPINotifier {
	return &TitlisAPINotifier{
		baseURL:        strings.TrimRight(baseURL, "/"),
		internalSecret: internalSecret,
		client:         &http.Client{Timeout: 10 * time.Second},
	}
}

// SendScorecardEvaluated posts a scorecard_evaluated event to titlis-api.
// Fire-and-forget: call from a goroutine. Failures are logged but not propagated.
func (n *TitlisAPINotifier) SendScorecardEvaluated(ctx context.Context, result scoring.ScoreResult) {
	payload := buildScorecardEvaluatedPayload(result)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("notifier: marshal envelope", "err", err, "uid", result.WorkloadUID)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.baseURL+"/v1/internal/scoreops/scorecard-evaluated", bytes.NewReader(body))
	if err != nil {
		slog.Error("notifier: build request", "err", err, "uid", result.WorkloadUID)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", n.internalSecret)

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Warn("notifier: send failed", "err", err, "uid", result.WorkloadUID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("notifier: unexpected status", "status", resp.StatusCode, "uid", result.WorkloadUID)
		return
	}
	slog.Debug("notifier: scorecard_evaluated sent", "uid", result.WorkloadUID, "status", resp.StatusCode)
}

// pillarForRule derives the pillar slug from the rule ID prefix.
func pillarForRule(ruleID string) string {
	switch {
	case strings.HasPrefix(ruleID, "RES"):
		return "resilience"
	case strings.HasPrefix(ruleID, "SEC"):
		return "security"
	case strings.HasPrefix(ruleID, "PERF"):
		return "performance"
	case strings.HasPrefix(ruleID, "OPS"):
		return "operational"
	case strings.HasPrefix(ruleID, "OBS"):
		return "observability"
	default:
		return "unknown"
	}
}

func buildScorecardEvaluatedPayload(r scoring.ScoreResult) map[string]any {
	pillarScores := make([]map[string]any, 0, len(r.PillarScores))
	for _, ps := range r.PillarScores {
		pillarScores = append(pillarScores, map[string]any{
			"pillar":        ps.Pillar,
			"score":         ps.Score,
			"passed_checks": ps.PassedChecks,
			"failed_checks": ps.TotalChecks - ps.PassedChecks,
			"weighted_score": ps.WeightedScore,
		})
	}

	validationResults := make([]map[string]any, 0, len(r.Findings))
	for _, f := range r.Findings {
		vr := map[string]any{
			"rule_id":       f.RuleID,
			"rule_name":     f.RuleName,
			"pillar":        pillarForRule(f.RuleID),
			"passed":        f.Passed,
			"severity":      f.Severity,
			"rule_type":     "binary",
			"weight":        f.Weight,
			"message":       f.Message,
			"is_remediable": f.IsRemediable,
		}
		if f.ActualValue != "" {
			vr["actual_value"] = f.ActualValue
		}
		validationResults = append(validationResults, vr)
	}

	payload := map[string]any{
		"workload_id":        r.WorkloadUID,
		"tenant_id":          r.TenantID,
		"namespace":          r.Namespace,
		"workload":           r.WorkloadName,
		"cluster":            r.Cluster,
		"environment":        "unknown",
		"k8s_event_type":     "MODIFIED",
		"overall_score":      r.OverallScore,
		"compliance_status":  r.ComplianceStatus,
		"total_rules":        r.TotalChecks,
		"passed_rules":       r.PassedChecks,
		"failed_rules":       r.TotalChecks - r.PassedChecks,
		"critical_failures":  r.CriticalIssues,
		"error_count":        r.ErrorIssues,
		"warning_count":      r.WarningIssues,
		"scorecard_version":  1,
		"workload_kind":      "Deployment",
		"pillar_scores":      pillarScores,
		"validation_results": validationResults,
		"evaluated_at":       r.CalculatedAt.Format(time.RFC3339),
	}
	if r.BackstageComponent != "" {
		payload["backstage_component"] = r.BackstageComponent
	}
	return payload
}

// SendQueueEvaluated posts a queue_evaluated event to titlis-api.
// Fire-and-forget: call from a goroutine. Failures are logged but not propagated.
func (n *TitlisAPINotifier) SendQueueEvaluated(ctx context.Context, result scoring.QueueScoreResult) {
	payload := buildQueueEvaluatedPayload(result)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("notifier: marshal queue envelope", "err", err, "external_id", result.ExternalID)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.baseURL+"/v1/internal/scoreops/queue-evaluated", bytes.NewReader(body))
	if err != nil {
		slog.Error("notifier: build queue request", "err", err, "external_id", result.ExternalID)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", n.internalSecret)

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Warn("notifier: queue send failed", "err", err, "external_id", result.ExternalID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		slog.Warn("notifier: queue unexpected status",
			"status", resp.StatusCode,
			"external_id", result.ExternalID,
			"body", string(b),
		)
		return
	}
	slog.Info("notifier: queue_evaluated sent", "external_id", result.ExternalID, "score", result.OverallScore, "status", resp.StatusCode)
}

func buildQueueEvaluatedPayload(r scoring.QueueScoreResult) map[string]any {
	pillarScores := make([]map[string]any, 0, len(r.PillarScores))
	for _, ps := range r.PillarScores {
		pillarScores = append(pillarScores, map[string]any{
			"pillar":       ps.Pillar,
			"pillarScore":  ps.Score,
			"passedChecks": ps.PassedChecks,
			"failedChecks": ps.TotalChecks - ps.PassedChecks,
			"weightedScore": ps.WeightedScore,
		})
	}

	validationResults := make([]map[string]any, 0, len(r.Findings))
	for _, f := range r.Findings {
		vr := map[string]any{
			"ruleId":        f.RuleID,
			"ruleName":      f.RuleName,
			"pillar":        queuePillarForRule(f.RuleID),
			"severity":      f.Severity,
			"rulePassed":    f.Passed,
			"resultMessage": f.Message,
		}
		if f.ActualValue != "" {
			vr["actualValue"] = f.ActualValue
		}
		validationResults = append(validationResults, vr)
	}

	return map[string]any{
		"provider":         r.Provider,
		"externalId":       r.ExternalID,
		"tenantId":         r.TenantID,
		"overallScore":     r.OverallScore,
		"complianceStatus": r.ComplianceStatus,
		"totalRules":       r.TotalChecks,
		"passedRules":      r.PassedChecks,
		"failedRules":      r.TotalChecks - r.PassedChecks,
		"criticalFailures": r.CriticalIssues,
		"errorCount":       r.ErrorIssues,
		"warningCount":     r.WarningIssues,
		"pillarScores":     pillarScores,
		"validationResults": validationResults,
		"evaluatedAt":      r.CalculatedAt.Format(time.RFC3339),
	}
}

func queuePillarForRule(ruleID string) string {
	switch {
	case len(ruleID) >= 2 && ruleID[:2] == "QR":
		return "resilience"
	case len(ruleID) >= 2 && ruleID[:2] == "QS":
		return "security"
	case len(ruleID) >= 2 && ruleID[:2] == "QP":
		return "performance"
	case len(ruleID) >= 2 && ruleID[:2] == "QO":
		return "operational"
	default:
		return "observability"
	}
}

// Ensure TitlisAPINotifier satisfies the interfaces expected by handlers.
var _ interface {
	SendScorecardEvaluated(ctx context.Context, result scoring.ScoreResult)
} = (*TitlisAPINotifier)(nil)

var _ interface {
	SendQueueEvaluated(ctx context.Context, result scoring.QueueScoreResult)
} = (*TitlisAPINotifier)(nil)

// NoopNotifier is used when SCOREOPS_TITLISAPI_URL is not configured.
type NoopNotifier struct{}

func (NoopNotifier) SendScorecardEvaluated(_ context.Context, result scoring.ScoreResult) {
	slog.Debug("notifier: noop — titlis-api URL not configured", "uid", result.WorkloadUID)
}

func (NoopNotifier) SendQueueEvaluated(_ context.Context, result scoring.QueueScoreResult) {
	slog.Debug("notifier: noop queue — titlis-api URL not configured", "external_id", result.ExternalID)
}

// ScorecardNotifier is the interface consumed by handlers.
type ScorecardNotifier interface {
	SendScorecardEvaluated(ctx context.Context, result scoring.ScoreResult)
}

// QueueNotifier is the interface consumed by the queue evaluate handler.
type QueueNotifier interface {
	SendQueueEvaluated(ctx context.Context, result scoring.QueueScoreResult)
}

