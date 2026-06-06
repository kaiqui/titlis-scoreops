package pillar

import (
	"strings"

	"github.com/titlis/scoreops/internal/scoring"
)

type OperationalPillar struct{}

func NewOperationalPillar() *OperationalPillar { return &OperationalPillar{} }

func (p *OperationalPillar) Slug() string { return "operational" }

func (p *OperationalPillar) RuleIDs() []string {
	return []string{"OPS-001", "OPS-002"}
}

func (p *OperationalPillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	var results []scoring.RuleResult
	if active["OPS-001"] {
		results = append(results, checkOPS001(snap))
	}
	if active["OPS-002"] {
		results = append(results, checkOPS002(snap))
	}
	return results
}

func checkOPS002(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	passed := snap.BackstageComponent != ""
	msg := "✅ Serviço registrado no catálogo do Backstage"
	if !passed {
		msg = "❌ Serviço não encontrado no catálogo do Backstage"
	}
	return ruleResult("OPS-002", "Backstage Registration", "error", 7.0, false, passed, msg, snap.BackstageComponent)
}

func checkOPS001(snap scoring.WorkloadSnapshot) scoring.RuleResult {
	ddLabels := []string{
		"tags.datadoghq.com/env",
		"tags.datadoghq.com/service",
		"tags.datadoghq.com/version",
	}

	var missing []string
	labels := snap.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	for _, l := range ddLabels {
		if labels[l] == "" {
			missing = append(missing, "labels["+l+"]")
		}
	}
	if labels["admission.datadoghq.com/enabled"] != "true" {
		missing = append(missing, "labels[admission.datadoghq.com/enabled=true]")
	}

	passed := len(missing) == 0
	msg := "✅ Instrumentação Datadog configurada corretamente"
	if !passed {
		msg = "❌ Configurações ausentes/inválidas: " + strings.Join(missing, ", ")
	}
	return ruleResult("OPS-001", "Datadog Labels", "warning", 8.0, false, passed, msg, "")
}
