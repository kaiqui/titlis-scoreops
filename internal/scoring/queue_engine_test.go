package scoring_test

import (
	"testing"
	"time"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

type stubQueuePillar struct {
	slug    string
	ruleIDs []string
	results []scoring.RuleResult
}

func (s *stubQueuePillar) Slug() string { return s.slug }
func (s *stubQueuePillar) RuleIDs() []string { return s.ruleIDs }
func (s *stubQueuePillar) Evaluate(_ domain.QueueSnapshot, _ map[string]bool, _ domain.QueueThresholds, _ domain.LabelRegistry) []scoring.RuleResult {
	return s.results
}

func TestQueueScoreEngine_Evaluate(t *testing.T) {
	engine := scoring.NewQueueScoreEngine()

	engine.RegisterPillar(&stubQueuePillar{
		slug:    "resilience",
		ruleIDs: []string{"QR-001", "QR-002"},
		results: []scoring.RuleResult{
			{RuleID: "QR-001", Passed: true, Weight: 10.0, Severity: "error"},
			{RuleID: "QR-002", Passed: false, Weight: 10.0, Severity: "error"},
		},
	})
	engine.RegisterPillar(&stubQueuePillar{
		slug:    "security",
		ruleIDs: []string{"QS-001"},
		results: []scoring.RuleResult{
			{RuleID: "QS-001", Passed: true, Weight: 7.0, Severity: "warning"},
		},
	})

	snap := domain.QueueSnapshot{
		Provider:    "gcp_pubsub",
		ExternalID:  "projects/proj/subscriptions/my-sub",
		TenantID:    1,
		CollectedAt: time.Now(),
	}

	active := engine.AllRulesActive()
	result := engine.Evaluate(snap, active, domain.QueueThresholds{}, domain.LabelRegistry{}, nil)

	if result.Provider != "gcp_pubsub" {
		t.Errorf("expected provider gcp_pubsub, got %s", result.Provider)
	}
	if result.TenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", result.TenantID)
	}
	if result.TotalChecks != 3 {
		t.Errorf("expected 3 total checks, got %d", result.TotalChecks)
	}
	if result.PassedChecks != 2 {
		t.Errorf("expected 2 passed checks, got %d", result.PassedChecks)
	}
	if result.ErrorIssues != 1 {
		t.Errorf("expected 1 error issue, got %d", result.ErrorIssues)
	}
	if len(result.PillarScores) != 2 {
		t.Errorf("expected 2 pillar scores, got %d", len(result.PillarScores))
	}

	// resilience: 1 of 2 passed → 50%
	for _, ps := range result.PillarScores {
		if ps.Pillar == "resilience" && ps.Score != 50.0 {
			t.Errorf("resilience score: expected 50.0, got %.2f", ps.Score)
		}
		if ps.Pillar == "security" && ps.Score != 100.0 {
			t.Errorf("security score: expected 100.0, got %.2f", ps.Score)
		}
	}
}

func TestQueueScoreEngine_AllRulesActive(t *testing.T) {
	engine := scoring.NewQueueScoreEngine()
	engine.RegisterPillar(&stubQueuePillar{
		slug:    "resilience",
		ruleIDs: []string{"QR-001", "QR-002"},
	})
	engine.RegisterPillar(&stubQueuePillar{
		slug:    "security",
		ruleIDs: []string{"QS-001"},
	})

	active := engine.AllRulesActive()
	if !active["QR-001"] || !active["QR-002"] || !active["QS-001"] {
		t.Error("AllRulesActive should return true for all registered rules")
	}
	if len(active) != 3 {
		t.Errorf("expected 3 rules, got %d", len(active))
	}
}
