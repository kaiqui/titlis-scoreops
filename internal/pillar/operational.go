package pillar

import (
	"strings"

	"github.com/titlis/scoreops/internal/scoring"
)

type OperationalPillar struct{}

func NewOperationalPillar() *OperationalPillar { return &OperationalPillar{} }

func (p *OperationalPillar) Slug() string { return "operational" }

func (p *OperationalPillar) RuleIDs() []string {
	return []string{"OPS-001"}
}

func (p *OperationalPillar) Evaluate(snap scoring.WorkloadSnapshot, active map[string]bool) []scoring.RuleResult {
	if active["OPS-001"] {
		return []scoring.RuleResult{checkOPS001(snap)}
	}
	return nil
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
