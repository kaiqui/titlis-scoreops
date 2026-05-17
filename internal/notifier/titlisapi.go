package notifier

import (
	"bytes"
	"context"
	"encoding/json"
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

	envelope := map[string]any{
		"v":    1,
		"t":    "scorecard_evaluated",
		"ts":   result.CalculatedAt.UnixMilli(),
		"data": payload,
	}

	body, err := json.Marshal(envelope)
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

	return map[string]any{
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
}

// Ensure TitlisAPINotifier satisfies the interface expected by handlers.
var _ interface {
	SendScorecardEvaluated(ctx context.Context, result scoring.ScoreResult)
} = (*TitlisAPINotifier)(nil)

// NoopNotifier is used when SCOREOPS_TITLISAPI_URL is not configured.
type NoopNotifier struct{}

func (NoopNotifier) SendScorecardEvaluated(_ context.Context, result scoring.ScoreResult) {
	slog.Debug("notifier: noop — titlis-api URL not configured", "uid", result.WorkloadUID)
}

// ScorecardNotifier is the interface consumed by handlers.
type ScorecardNotifier interface {
	SendScorecardEvaluated(ctx context.Context, result scoring.ScoreResult)
}

