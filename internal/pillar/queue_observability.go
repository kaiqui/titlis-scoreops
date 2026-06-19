package pillar

import (
	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

type QueueObservabilityPillar struct{}

func NewQueueObservabilityPillar() *QueueObservabilityPillar { return &QueueObservabilityPillar{} }

func (p *QueueObservabilityPillar) Slug() string { return "observability" }

func (p *QueueObservabilityPillar) RuleIDs() []string {
	return []string{"QObs-001", "QObs-002", "QObs-003", "QObs-004"}
}

func (p *QueueObservabilityPillar) Evaluate(
	snap      domain.QueueSnapshot,
	active    map[string]bool,
	thresholds domain.QueueThresholds,
	registry  domain.LabelRegistry,
) []scoring.RuleResult {
	type check struct {
		id string
		fn func() scoring.RuleResult
	}

	checks := []check{
		{"QObs-001", func() scoring.RuleResult { return checkQObs001(snap) }},
		{"QObs-002", func() scoring.RuleResult { return checkQObs002(snap) }},
		{"QObs-003", func() scoring.RuleResult { return checkQObs003(snap) }},
		{"QObs-004", func() scoring.RuleResult { return checkQObs004(snap) }},
	}

	var results []scoring.RuleResult
	for _, c := range checks {
		if !active[c.id] {
			continue
		}
		results = append(results, c.fn())
	}
	return results
}

func checkQObs001(snap domain.QueueSnapshot) scoring.RuleResult {
	passed := snap.HasMonitorBacklog
	msg := "✅ Monitor de backlog configurado"
	if !passed {
		msg = "❌ Monitor de backlog ausente — alertas de acúmulo não serão disparados"
	}
	return ruleResult("QObs-001", "Monitor de backlog configurado", "warning", 7.0, true, passed, msg, "")
}

func checkQObs002(snap domain.QueueSnapshot) scoring.RuleResult {
	passed := snap.HasMonitorAge
	msg := "✅ Monitor de age configurado"
	if !passed {
		msg = "❌ Monitor de age ausente — mensagens atrasadas não serão alertadas"
	}
	return ruleResult("QObs-002", "Monitor de age configurado", "warning", 7.0, true, passed, msg, "")
}

func checkQObs003(snap domain.QueueSnapshot) scoring.RuleResult {
	if !snap.IsDLQ {
		return ruleResult("QObs-003", "Monitor de DLQ configurado", "error", 8.0, false, true,
			"✅ Não é uma DLQ — regra não aplicável", "")
	}
	passed := snap.HasMonitorDLQ
	msg := "✅ Monitor de DLQ configurado"
	if !passed {
		msg = "❌ Monitor de DLQ ausente — saturação da dead-letter queue não será alertada"
	}
	return ruleResult("QObs-003", "Monitor de DLQ configurado", "error", 8.0, true, passed, msg, "")
}

func checkQObs004(snap domain.QueueSnapshot) scoring.RuleResult {
	hasSubscriptionID := snap.ExternalID != ""
	hasTopicID        := snap.TopicID != ""
	hasProjectID      := snap.ProjectID != ""
	passed := hasSubscriptionID && hasTopicID && hasProjectID

	msg := "✅ Tags de correlação presentes: subscription_id, topic_id, project_id"
	if !passed {
		var missing []string
		if !hasSubscriptionID {
			missing = append(missing, "subscription_id")
		}
		if !hasTopicID {
			missing = append(missing, "topic_id")
		}
		if !hasProjectID {
			missing = append(missing, "project_id")
		}
		msg = "❌ Tags de correlação ausentes: "
		for i, m := range missing {
			if i > 0 {
				msg += ", "
			}
			msg += m
		}
	}
	return ruleResult("QObs-004", "Tags para correlação", "info", 3.0, false, passed, msg, "")
}
