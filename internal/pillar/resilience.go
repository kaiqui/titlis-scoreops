package pillar

import (
	"fmt"

	"github.com/titlis/scoreops/internal/scoring"
)

type ResiliencePillar struct{}

func NewResiliencePillar() *ResiliencePillar { return &ResiliencePillar{} }

func (p *ResiliencePillar) Slug() string { return "resilience" }

func (p *ResiliencePillar) RuleIDs() []string {
	return []string{
		"RES-001", "RES-002", "RES-003", "RES-004", "RES-005",
		"RES-006", "RES-007", "RES-008", "RES-009", "RES-010",
		"RES-011", "RES-012", "RES-013", "RES-014", "RES-016",
		"RES-017", "RES-018", "RES-019",
	}
}

func (p *ResiliencePillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	type check struct {
		id         string
		highOnly   bool
		fn         func() scoring.RuleResult
	}

	checks := []check{
		{"RES-001", false, func() scoring.RuleResult { return checkRES001(snap) }},
		{"RES-002", false, func() scoring.RuleResult { return checkRES002(snap) }},
		{"RES-003", false, func() scoring.RuleResult { return checkRES003(snap) }},
		{"RES-004", false, func() scoring.RuleResult { return checkRES004(snap) }},
		{"RES-005", false, func() scoring.RuleResult { return checkRES005(snap) }},
		{"RES-006", false, func() scoring.RuleResult { return checkRES006(snap) }},
		{"RES-007", false, func() scoring.RuleResult { return checkRES007(snap) }},
		{"RES-008", false, func() scoring.RuleResult { return checkRES008(snap) }},
		{"RES-009", false, func() scoring.RuleResult { return checkRES009(snap) }},
		{"RES-010", false, func() scoring.RuleResult { return checkRES010(snap) }},
		{"RES-011", false, func() scoring.RuleResult { return checkRES011(snap) }},
		{"RES-012", false, func() scoring.RuleResult { return checkRES012(snap) }},
		{"RES-013", false, func() scoring.RuleResult { return checkRES013(snap) }},
		{"RES-014", false, func() scoring.RuleResult { return checkRES014(snap) }},
		{"RES-016", false, func() scoring.RuleResult { return checkRES016(snap) }},
		{"RES-017", true, func() scoring.RuleResult { return checkRES017(snap) }},
		{"RES-018", true, func() scoring.RuleResult { return checkRES018(snap) }},
		{"RES-019", true, func() scoring.RuleResult { return checkRES019(snap) }},
	}

	var results []scoring.RuleResult
	for _, c := range checks {
		if !active[c.id] {
			continue
		}
		if c.highOnly && snap.Criticality != "high" {
			continue
		}
		results = append(results, c.fn())
	}
	return results
}

func checkRES001(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasLivenessProbe
	msg := "✅ Liveness probe configurado"
	if !passed {
		msg = "❌ Liveness probe não configurado"
	}
	return ruleResult("RES-001", "Liveness Probe", "error", 10.0, true, passed, msg, "")
}

func checkRES002(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasReadinessProbe
	msg := "✅ Readiness probe configurado"
	if !passed {
		msg = "❌ Readiness probe não configurado"
	}
	return ruleResult("RES-002", "Readiness Probe", "error", 10.0, true, passed, msg, "")
}

func checkRES003(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.CPURequestSet
	msg := "✅ CPU request definido"
	if !passed {
		msg = "❌ CPU request não definido"
	}
	return ruleResult("RES-003", "CPU Request", "error", 8.0, true, passed, msg, "")
}

func checkRES004(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.CPULimitSet
	msg := "✅ CPU limit definido"
	if !passed {
		msg = "❌ CPU limit não definido"
	}
	return ruleResult("RES-004", "CPU Limit", "warning", 5.0, true, passed, msg, "")
}

func checkRES005(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.MemoryRequestSet
	msg := "✅ Memory request definido"
	if !passed {
		msg = "❌ Memory request não definido"
	}
	return ruleResult("RES-005", "Memory Request", "error", 8.0, true, passed, msg, "")
}

func checkRES006(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.MemoryLimitSet
	msg := "✅ Memory limit definido"
	if !passed {
		msg = "❌ Memory limit não definido"
	}
	return ruleResult("RES-006", "Memory Limit", "warning", 5.0, true, passed, msg, "")
}

func checkRES007(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasHPA
	msg := "✅ HPA existe"
	if !passed {
		msg = "❌ HPA não encontrado para este Deployment"
	}
	return ruleResult("RES-007", "HPA Exists", "warning", 7.0, true, passed, msg, "")
}

func checkRES008(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasHPA && snap.HPAHasMetrics
	msg := "✅ HPA tem métricas configuradas"
	if !passed {
		msg = "❌ HPA sem métricas ou HPA ausente"
	}
	return ruleResult("RES-008", "HPA Has Metrics", "warning", 5.0, true, passed, msg, "")
}

func checkRES009(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	// TerminationGracePeriodSec == 0 means "not explicitly configured"
	passed := snap.TerminationGracePeriodSec > 0
	msg := "✅ TerminationGracePeriodSeconds configurado"
	actual := ""
	if !passed {
		msg = "❌ TerminationGracePeriodSeconds não configurado"
	} else {
		actual = fmt.Sprintf("%d", snap.TerminationGracePeriodSec)
	}
	return ruleResult("RES-009", "Termination Grace Period", "info", 3.0, false, passed, msg, actual)
}

func checkRES010(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.RunAsNonRoot
	msg := "✅ Container configurado para rodar como não-root"
	if !passed {
		msg = "❌ Container pode rodar como root (runAsNonRoot não está true)"
	}
	return ruleResult("RES-010", "Run As Non Root", "error", 10.0, false, passed, msg, "")
}

func checkRES011(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasPodSecurityContext
	msg := "✅ Pod Security Context configurado"
	if !passed {
		msg = "❌ Pod Security Context não configurado"
	}
	return ruleResult("RES-011", "Pod Security Context", "warning", 5.0, false, passed, msg, "")
}

func checkRES012(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasNetworkPolicy
	msg := "✅ NetworkPolicy existe no namespace"
	if !passed {
		msg = "❌ Nenhuma NetworkPolicy encontrada no namespace"
	}
	return ruleResult("RES-012", "Network Policy", "warning", 7.0, false, passed, msg, "")
}

func checkRES013(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	replicas := snap.Replicas
	if replicas == 0 {
		replicas = 1
	}
	passed := replicas >= 2
	msg := fmt.Sprintf("✅ Réplicas: %d", replicas)
	actual := fmt.Sprintf("%d", replicas)
	if !passed {
		msg = fmt.Sprintf("❌ Réplicas insuficientes: %d (mínimo: 2)", replicas)
	}
	return ruleResult("RES-013", "Minimum Replicas", "warning", 6.0, false, passed, msg, actual)
}

func checkRES014(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.Strategy != ""
	msg := "✅ Estratégia de deployment configurada"
	actual := snap.Strategy
	if !passed {
		msg = "❌ Estratégia de deployment não configurada"
		actual = ""
	}
	return ruleResult("RES-014", "Deployment Strategy", "warning", 4.0, false, passed, msg, actual)
}

func checkRES016(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasHPA {
		return ruleResult("RES-016", "HPA Min Replicas", "warning", 5.0, true, false,
			"❌ HPA ausente ou minReplicas não configurado", "")
	}
	passed := snap.HPAMinReplicas >= 2
	msg := fmt.Sprintf("✅ HPA minReplicas: %d", snap.HPAMinReplicas)
	actual := fmt.Sprintf("%d", snap.HPAMinReplicas)
	if !passed {
		msg = fmt.Sprintf("❌ HPA minReplicas insuficiente: %d (mínimo: 2)", snap.HPAMinReplicas)
	}
	return ruleResult("RES-016", "HPA Min Replicas", "warning", 5.0, true, passed, msg, actual)
}

// checkRES017 — high criticality only.
// HPAScaleUpStabilizationSec == 0 means fast scale-up (passes). -1 means not configured (fails).
func checkRES017(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasHPA || snap.HPAScaleUpStabilizationSec < 0 {
		return ruleResult("RES-017", "HPA ScaleUp Stabilization", "warning", 4.0, false, false,
			"❌ HPA scaleUp stabilization não configurado", "")
	}
	passed := snap.HPAScaleUpStabilizationSec == 0
	actual := fmt.Sprintf("%d", snap.HPAScaleUpStabilizationSec)
	msg := fmt.Sprintf("✅ HPA scaleUp stabilizationWindowSeconds: %d", snap.HPAScaleUpStabilizationSec)
	if !passed {
		msg = fmt.Sprintf("❌ HPA scaleUp stabilizationWindowSeconds muito alto: %d", snap.HPAScaleUpStabilizationSec)
	}
	return ruleResult("RES-017", "HPA ScaleUp Stabilization", "warning", 4.0, false, passed, msg, actual)
}

// checkRES018 — high criticality only.
// Passes if HPAScaleDownStabilizationSec >= 300 (safe scale-down delay).
func checkRES018(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasHPA || snap.HPAScaleDownStabilizationSec < 0 {
		return ruleResult("RES-018", "HPA ScaleDown Stabilization", "warning", 4.0, false, false,
			"❌ HPA scaleDown stabilization não configurado", "")
	}
	passed := snap.HPAScaleDownStabilizationSec >= 300
	actual := fmt.Sprintf("%d", snap.HPAScaleDownStabilizationSec)
	msg := fmt.Sprintf("✅ HPA scaleDown stabilizationWindowSeconds: %d", snap.HPAScaleDownStabilizationSec)
	if !passed {
		msg = fmt.Sprintf("❌ HPA scaleDown stabilizationWindowSeconds muito baixo: %d (mínimo: 300)", snap.HPAScaleDownStabilizationSec)
	}
	return ruleResult("RES-018", "HPA ScaleDown Stabilization", "warning", 4.0, false, passed, msg, actual)
}

// checkRES019 — high criticality only.
func checkRES019(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.HasHPA && snap.HPAHasBehaviorPolicies
	msg := "✅ HPA tem policies de behavior em scaleUp e scaleDown"
	if !passed {
		msg = "❌ HPA sem policies de behavior ou HPA ausente"
	}
	return ruleResult("RES-019", "HPA Behavior Policies", "warning", 4.0, false, passed, msg, "")
}
