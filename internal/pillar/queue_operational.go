package pillar

import (
	"fmt"
	"strings"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

type QueueOperationalPillar struct{}

func NewQueueOperationalPillar() *QueueOperationalPillar { return &QueueOperationalPillar{} }

func (p *QueueOperationalPillar) Slug() string { return "operational" }

func (p *QueueOperationalPillar) RuleIDs() []string {
	return []string{"QO-001", "QO-002", "QO-003", "QO-004"}
}

func (p *QueueOperationalPillar) Evaluate(
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
		{"QO-001", func() scoring.RuleResult { return checkQO001(snap) }},
		{"QO-002", func() scoring.RuleResult { return checkQO002(snap) }},
		{"QO-003", func() scoring.RuleResult { return checkQO003(snap) }},
		{"QO-004", func() scoring.RuleResult { return checkQO004(snap) }},
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

// subscriptionBaseName extracts the last component of a GCP Pub/Sub subscription path.
// "projects/proj/subscriptions/my-name" → "my-name"
func subscriptionBaseName(externalID string) string {
	parts := strings.Split(externalID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return externalID
}

func checkQO001(snap domain.QueueSnapshot) scoring.RuleResult {
	name := subscriptionBaseName(snap.ExternalID)
	if snap.DisplayName != "" {
		name = snap.DisplayName
	}
	passed := strings.HasSuffix(name, "-sub") || strings.HasSuffix(name, "-dlq")
	actual := name
	msg := fmt.Sprintf("✅ Nome segue convenção: %s", name)
	if !passed {
		msg = fmt.Sprintf("❌ Nome sem sufixo -sub ou -dlq: %s", name)
	}
	return ruleResult("QO-001", "Convenção de nomenclatura", "info", 4.0, false, passed, msg, actual)
}

func checkQO002(snap domain.QueueSnapshot) scoring.RuleResult {
	passed := snap.HasSnapshotPolicy
	msg := "✅ Snapshot configurado para esta subscription"
	if !passed {
		msg = "❌ Nenhum snapshot configurado — dados podem ser perdidos"
	}
	return ruleResult("QO-002", "Snapshot configurado", "info", 3.0, false, passed, msg, "")
}

func checkQO003(snap domain.QueueSnapshot) scoring.RuleResult {
	if !snap.IsDLQ {
		return ruleResult("QO-003", "DLQ drenada regularmente", "warning", 6.0, false, true,
			"✅ Não é uma DLQ — regra não aplicável", "")
	}
	actual := fmt.Sprintf("%d", snap.DeadLetterMessageCount)
	passed := snap.DeadLetterMessageCount == 0
	msg := "✅ DLQ vazia"
	if !passed {
		msg = fmt.Sprintf("❌ DLQ com %d mensagens pendentes — verificar se estão sendo drenadas", snap.DeadLetterMessageCount)
	}
	return ruleResult("QO-003", "DLQ drenada regularmente", "warning", 6.0, false, passed, msg, actual)
}

func checkQO004(snap domain.QueueSnapshot) scoring.RuleResult {
	passed := snap.SendMessageCountRate > 0
	actual := fmt.Sprintf("%.4f msg/s", snap.SendMessageCountRate)
	msg := fmt.Sprintf("✅ Subscription com throughput: %.4f msg/s", snap.SendMessageCountRate)
	if !passed {
		msg = "❌ Nenhuma mensagem publicada na última hora (send_rate = 0)"
	}
	return ruleResult("QO-004", "Subscription com throughput", "warning", 5.0, false, passed, msg, actual)
}
