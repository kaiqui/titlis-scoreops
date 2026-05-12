package scoring

import (
	"testing"
)

// stubPillar is a minimal PillarModule for engine tests.
type stubPillar struct {
	slug    string
	ruleIDs []string
	results []RuleResult
}

func (s *stubPillar) Slug() string    { return s.slug }
func (s *stubPillar) RuleIDs() []string { return s.ruleIDs }
func (s *stubPillar) Evaluate(_ WorkloadSnapshot, active map[string]bool) []RuleResult {
	var out []RuleResult
	for _, r := range s.results {
		if active[r.RuleID] {
			out = append(out, r)
		}
	}
	return out
}

func allActive(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func TestScoreEngine_AllPassed(t *testing.T) {
	engine := NewScoreEngine()
	p := &stubPillar{
		slug:    "resilience",
		ruleIDs: []string{"RES-001", "RES-002"},
		results: []RuleResult{
			{RuleID: "RES-001", Passed: true, Severity: "error", Weight: 10},
			{RuleID: "RES-002", Passed: true, Severity: "error", Weight: 10},
		},
	}
	engine.RegisterPillar(p)

	snap := WorkloadSnapshot{UID: "u1", Name: "app", Namespace: "default", Cluster: "c1", TenantID: 1, EngineSlug: "kubernetes"}
	active := allActive(p.ruleIDs)
	weights := map[string]float64{"resilience": 40}

	result := engine.Evaluate(snap, active, weights)

	if result.OverallScore != 100 {
		t.Errorf("expected 100, got %.2f", result.OverallScore)
	}
	if result.ComplianceStatus != "COMPLIANT" {
		t.Errorf("expected COMPLIANT, got %s", result.ComplianceStatus)
	}
	if result.PassedChecks != 2 || result.TotalChecks != 2 {
		t.Errorf("expected 2/2 checks passed, got %d/%d", result.PassedChecks, result.TotalChecks)
	}
}

func TestScoreEngine_AllFailed(t *testing.T) {
	engine := NewScoreEngine()
	p := &stubPillar{
		slug:    "security",
		ruleIDs: []string{"SEC-001"},
		results: []RuleResult{
			{RuleID: "SEC-001", Passed: false, Severity: "error", Weight: 9},
		},
	}
	engine.RegisterPillar(p)

	snap := WorkloadSnapshot{UID: "u1", Name: "app", Namespace: "default", Cluster: "c1", TenantID: 1, EngineSlug: "kubernetes"}
	active := allActive(p.ruleIDs)
	weights := map[string]float64{"security": 30}

	result := engine.Evaluate(snap, active, weights)

	if result.OverallScore != 0 {
		t.Errorf("expected 0, got %.2f", result.OverallScore)
	}
	if result.ComplianceStatus != "NON_COMPLIANT" {
		t.Errorf("expected NON_COMPLIANT, got %s", result.ComplianceStatus)
	}
	if result.ErrorIssues != 1 {
		t.Errorf("expected 1 error issue, got %d", result.ErrorIssues)
	}
}

func TestScoreEngine_InactiveRulesSkipped(t *testing.T) {
	engine := NewScoreEngine()
	p := &stubPillar{
		slug:    "resilience",
		ruleIDs: []string{"RES-001", "RES-002"},
		results: []RuleResult{
			{RuleID: "RES-001", Passed: true, Weight: 10},
			{RuleID: "RES-002", Passed: false, Weight: 10},
		},
	}
	engine.RegisterPillar(p)

	snap := WorkloadSnapshot{UID: "u1", Name: "app", Namespace: "ns", Cluster: "c1", TenantID: 1, EngineSlug: "kubernetes"}
	// Only RES-001 is active; RES-002 is disabled
	active := map[string]bool{"RES-001": true, "RES-002": false}
	weights := map[string]float64{"resilience": 40}

	result := engine.Evaluate(snap, active, weights)

	if result.TotalChecks != 1 {
		t.Errorf("expected 1 total check (RES-002 excluded), got %d", result.TotalChecks)
	}
	if result.OverallScore != 100 {
		t.Errorf("expected 100 score with only passed rule active, got %.2f", result.OverallScore)
	}
}

func TestScoreEngine_WeightedAverage(t *testing.T) {
	engine := NewScoreEngine()

	// resilience: 100% score, weight 40
	res := &stubPillar{
		slug:    "resilience",
		ruleIDs: []string{"RES-001"},
		results: []RuleResult{{RuleID: "RES-001", Passed: true, Weight: 10}},
	}
	// security: 0% score, weight 30
	sec := &stubPillar{
		slug:    "security",
		ruleIDs: []string{"SEC-001"},
		results: []RuleResult{{RuleID: "SEC-001", Passed: false, Weight: 9}},
	}
	engine.RegisterPillar(res)
	engine.RegisterPillar(sec)

	snap := WorkloadSnapshot{UID: "u1", Name: "app", Namespace: "ns", Cluster: "c1", TenantID: 1, EngineSlug: "kubernetes"}
	active := map[string]bool{"RES-001": true, "SEC-001": true}
	weights := map[string]float64{"resilience": 40, "security": 30}

	result := engine.Evaluate(snap, active, weights)

	// expected: (100*40 + 0*30) / (40+30) = 4000/70 ≈ 57.14
	expected := (100.0 * 40) / (40 + 30)
	diff := result.OverallScore - expected
	if diff < -0.01 || diff > 0.01 {
		t.Errorf("expected overall score %.2f, got %.2f", expected, result.OverallScore)
	}
	if result.ComplianceStatus != "NON_COMPLIANT" {
		t.Errorf("expected NON_COMPLIANT at score %.2f", result.OverallScore)
	}
}
