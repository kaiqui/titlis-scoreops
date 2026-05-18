package pillar

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/titlis/scoreops/internal/insights"
	"github.com/titlis/scoreops/internal/scoring"
)

type PerformancePillar struct {
	ins insights.Client
}

func NewPerformancePillar(ins insights.Client) *PerformancePillar {
	return &PerformancePillar{ins: ins}
}

func (p *PerformancePillar) Slug() string { return "performance" }

func (p *PerformancePillar) RuleIDs() []string {
	return []string{"PERF-001", "PERF-002", "PERF-003", "PERF-004"}
}

func (p *PerformancePillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	checks := []struct {
		id string
		fn func() scoring.RuleResult
	}{
		{"PERF-001", func() scoring.RuleResult { return checkPERF001(snap) }},
		{"PERF-002", func() scoring.RuleResult { return checkPERF002(snap) }},
		{"PERF-003", func() scoring.RuleResult { return checkPERF003(snap) }},
		{"PERF-004", func() scoring.RuleResult { return checkPERF004(snap, p.ins) }},
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
	passed := target <= 70
	actual := fmt.Sprintf("%d", target)
	msg := fmt.Sprintf("✅ HPA CPU target: %d%%", target)
	if !passed {
		msg = fmt.Sprintf("❌ HPA CPU target muito alto: %d%% (máximo: 70%%)", target)
	}
	return ruleResult("PERF-003", "HPA CPU Target Ceiling", "info", 3.0, true, passed, msg, actual)
}

// checkPERF004 verifies that HPA values are aligned with real workload data (from titlis-insights)
// or with the configured environment template when Datadog is not available.
// Returns "skipped" when neither data source is available — this is not a failure.
func checkPERF004(snap scoring.WorkloadSnapshot, ins insights.Client) scoring.RuleResult {
	if ins == nil {
		return ruleResult("PERF-004", "HPA values aligned with workload data", "warning", 6.0, true, false,
			"⏭ Skipped: insights não configurado", "skipped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reco, err := ins.GetHpaRecommendation(ctx, insights.RecommendationRequest{
		TenantID:    snap.TenantID,
		WorkloadUID: snap.UID,
		Environment: snap.Environment,
		Criticality: snap.Criticality,
		HasDatadog:  snap.HasDatadog,
	})
	if err != nil {
		return ruleResult("PERF-004", "HPA values aligned with workload data", "warning", 6.0, true, false,
			fmt.Sprintf("⏭ Skipped: erro ao consultar insights: %v", err), "errored")
	}
	if reco.Source == "skipped" {
		return ruleResult("PERF-004", "HPA values aligned with workload data", "warning", 6.0, true, false,
			fmt.Sprintf("⏭ Skipped: %s", reco.Notes), "skipped")
	}

	if !snap.HasHPA {
		return ruleResult("PERF-004", "HPA values aligned with workload data", "warning", 6.0, true, false,
			"❌ HPA ausente — valores recomendados não aplicados", "no_hpa")
	}

	const tolerancePct = 20
	minOK := withinTolerance(snap.HPAMinReplicas, reco.MinReplicas, tolerancePct)
	maxOK := withinTolerance(snap.HPAMaxReplicas, reco.MaxReplicas, tolerancePct)
	cpuOK := snap.HPACPUTargetPercent == 0 || withinTolerance(snap.HPACPUTargetPercent, reco.TargetCPUPct, tolerancePct)

	passed := minOK && maxOK && cpuOK
	actual := fmt.Sprintf("min:%d max:%d cpu:%d%%", snap.HPAMinReplicas, snap.HPAMaxReplicas, snap.HPACPUTargetPercent)
	expected := fmt.Sprintf("min:%d max:%d cpu:%d%%", reco.MinReplicas, reco.MaxReplicas, reco.TargetCPUPct)
	msg := fmt.Sprintf("✅ HPA alinhado com dados reais (fonte: %s, confiança: %.0f%%)", reco.Source, reco.Confidence*100)
	if !passed {
		msg = fmt.Sprintf("❌ HPA desalinhado — esperado: %s, atual: %s (fonte: %s)", expected, actual, reco.Source)
	}
	return ruleResult("PERF-004", "HPA values aligned with workload data", "warning", 6.0, true, passed, msg, actual)
}

// withinTolerance returns true if |actual-expected|/expected <= pct/100.
func withinTolerance(actual, expected, pct int) bool {
	if expected == 0 {
		return actual == 0
	}
	delta := math.Abs(float64(actual-expected)) / float64(expected) * 100
	return delta <= float64(pct)
}
