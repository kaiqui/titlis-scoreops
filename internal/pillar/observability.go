package pillar

import "github.com/titlis/scoreops/internal/scoring"

// ObservabilityPillar evaluates SLO coverage for workloads tracked in Datadog.
// Rules are skipped entirely (not returned) when has_datadog=false, so workloads without
// Datadog do not receive score penalties — the pillar weight is excluded from totalWeight
// in that case (see engine.go: skip pillar when len(results) == 0).
type ObservabilityPillar struct{}

func NewObservabilityPillar() *ObservabilityPillar { return &ObservabilityPillar{} }

func (p *ObservabilityPillar) Slug() string { return "observability" }

func (p *ObservabilityPillar) RuleIDs() []string {
	return []string{"OBS-001", "OBS-002"}
}

func (p *ObservabilityPillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	// OBS-001 (SLO existence) is meaningful regardless of Datadog — a workload should have
	// a declared SLO even before Datadog monitoring is active.
	//
	// OBS-002 (Datadog sync) requires an active Datadog integration to evaluate. When
	// has_datadog=false the rule is returned as exempt (passed=true) so it is visible in the
	// UI with a clear "not applicable" message without penalising the score.
	var results []scoring.RuleResult
	if active["OBS-001"] {
		results = append(results, checkOBS001(snap))
	}
	if active["OBS-002"] {
		if !snap.HasDatadog {
			results = append(results, ruleResult(
				"OBS-002", "SLO Sincronizado", "warning", 6.0, false, true,
				"⏭ Sincronização com Datadog não verificada — adicione a label `tags.datadoghq.com/service` para ativar esta verificação", "",
			))
		} else {
			results = append(results, checkOBS002(snap))
		}
	}
	return results
}

// checkOBS001 verifica se existe ao menos um SLO configurado para o namespace do workload.
func checkOBS001(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if snap.HasSLO {
		return ruleResult(
			"OBS-001", "SLO Configurado", "warning", 8.0, false, true,
			"✅ SLO configurado para este serviço", "",
		)
	}
	return ruleResult(
		"OBS-001", "SLO Configurado", "warning", 8.0, false, false,
		"❌ Nenhum SLO configurado — crie um CRD SLOConfig no cluster para este namespace", "",
	)
}

// checkOBS002 verifica se o SLO está sincronizado com o Datadog e em estado saudável.
// Skipped (not returned as failure) when no SLO exists — OBS-001 already covers that case.
func checkOBS002(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	if !snap.HasSLO {
		// No SLO to evaluate health for; OBS-001 covers the missing-SLO case.
		// Return a neutral result so the weight is still counted but not as a double-penalty.
		return ruleResult(
			"OBS-002", "SLO Sincronizado", "warning", 6.0, false, false,
			"⏭ SLO não configurado — configure um SLOConfig antes de avaliar sincronização", "no_slo",
		)
	}
	if snap.SLOHealthy {
		return ruleResult(
			"OBS-002", "SLO Sincronizado", "warning", 6.0, false, true,
			"✅ SLO sincronizado com Datadog e em estado saudável", "",
		)
	}
	return ruleResult(
		"OBS-002", "SLO Sincronizado", "warning", 6.0, false, false,
		"❌ SLO com erro de sincronização com o Datadog — verifique o estado do SLOConfig no cluster", "",
	)
}
