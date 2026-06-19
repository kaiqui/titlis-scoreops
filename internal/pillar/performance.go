package pillar

import (
	"fmt"

	"github.com/titlis/scoreops/internal/scoring"
)

type PerformancePillar struct{}

func NewPerformancePillar() *PerformancePillar {
	return &PerformancePillar{}
}

func (p *PerformancePillar) Slug() string { return "performance" }

func (p *PerformancePillar) RuleIDs() []string {
	return []string{"PERF-001", "PERF-002", "PERF-003"}
}

func (p *PerformancePillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	checks := []struct {
		id string
		fn func() scoring.RuleResult
	}{
		{"PERF-001", func() scoring.RuleResult { return checkPERF001(snap) }},
		{"PERF-002", func() scoring.RuleResult { return checkPERF002(snap) }},
		{"PERF-003", func() scoring.RuleResult { return checkPERF003(snap) }},
	}

	var results []scoring.RuleResult
	for _, c := range checks {
		if active[c.id] {
			results = append(results, c.fn())
		}
	}
	return results
}

func checkPERF001(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.CPURequestSet || !snap.CPULimitSet {
		return ruleResult("PERF-001", "CPU Limit Ratio", "warning", 4.0, false, false,
			"❌ CPU request ou limit não definido — não é possível calcular ratio", "")
	}
	ratio := snap.CPULimitRatio
	passed := ratio > 0 && ratio <= 3.0
	actual := fmt.Sprintf("%.2f", ratio)
	msg := fmt.Sprintf("✅ CPU limit/request ratio: %.1f", ratio)
	if !passed {
		msg = fmt.Sprintf("❌ CPU limit/request ratio muito alto: %.1f (máximo: 3.0)", ratio)
	}
	return ruleResult("PERF-001", "CPU Limit Ratio", "warning", 4.0, false, passed, msg, actual)
}

func checkPERF002(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasHPA || snap.HPACPUTargetPercent == 0 {
		return ruleResult("PERF-002", "HPA CPU Target Range", "info", 3.0, true, false,
			"❌ HPA sem target de CPU ou HPA ausente", "")
	}
	target := snap.HPACPUTargetPercent
	passed := target >= 50 && target <= 90
	actual := fmt.Sprintf("%d", target)
	msg := fmt.Sprintf("✅ HPA CPU target: %d%%", target)
	if !passed {
		msg = fmt.Sprintf("❌ HPA CPU target fora do range: %d%% (esperado: 50-90%%)", target)
	}
	return ruleResult("PERF-002", "HPA CPU Target Range", "info", 3.0, true, passed, msg, actual)
}

func checkPERF003(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasHPA || snap.HPACPUTargetPercent == 0 {
		return ruleResult("PERF-003", "HPA CPU Target Ceiling", "info", 3.0, true, false,
			"❌ HPA sem target de CPU ou HPA ausente", "")
	}
	target := snap.HPACPUTargetPercent
	passed := target <= 90
	actual := fmt.Sprintf("%d", target)
	msg := fmt.Sprintf("✅ HPA CPU target: %d%%", target)
	if !passed {
		msg = fmt.Sprintf("❌ HPA CPU target muito alto: %d%% (máximo: 90%%)", target)
	}
	return ruleResult("PERF-003", "HPA CPU Target Ceiling", "info", 3.0, true, passed, msg, actual)
}

