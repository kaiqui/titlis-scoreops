package pillar

import (
	"fmt"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

type QueuePerformancePillar struct{}

func NewQueuePerformancePillar() *QueuePerformancePillar { return &QueuePerformancePillar{} }

func (p *QueuePerformancePillar) Slug() string { return "performance" }

func (p *QueuePerformancePillar) RuleIDs() []string {
	return []string{"QP-001", "QP-002", "QP-003"}
}

func (p *QueuePerformancePillar) Evaluate(
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
		{"QP-001", func() scoring.RuleResult { return checkQP001(snap) }},
		{"QP-002", func() scoring.RuleResult { return checkQP002(snap) }},
		{"QP-003", func() scoring.RuleResult { return checkQP003(snap, thresholds) }},
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

func checkQP001(snap domain.QueueSnapshot) scoring.RuleResult {
	passed := snap.PullMessageCountRate > 0
	actual := fmt.Sprintf("%.4f msg/s", snap.PullMessageCountRate)
	msg := fmt.Sprintf("✅ Consumidor ativo: %.4f msg/s", snap.PullMessageCountRate)
	if !passed {
		msg = "❌ Nenhum consumidor ativo detectado (pull_rate = 0)"
	}
	return ruleResult("QP-001", "Consumidor ativo", "error", 9.0, true, passed, msg, actual)
}

func checkQP002(snap domain.QueueSnapshot) scoring.RuleResult {
	if snap.SendMessageCountRate == 0 {
		return ruleResult("QP-002", "Taxa de processamento saudável", "warning", 6.0, false, true,
			"✅ Sem produção de mensagens — taxa de processamento não aplicável", "")
	}
	ratio := snap.AckMessageCountRate / snap.SendMessageCountRate
	passed := ratio >= 0.80
	actual := fmt.Sprintf("%.2f%%", ratio*100)
	msg := fmt.Sprintf("✅ Taxa de processamento: %.2f%% (ack/send)", ratio*100)
	if !passed {
		msg = fmt.Sprintf("❌ Taxa de processamento baixa: %.2f%% (mínimo: 80%%)", ratio*100)
	}
	return ruleResult("QP-002", "Taxa de processamento saudável", "warning", 6.0, false, passed, msg, actual)
}

func checkQP003(snap domain.QueueSnapshot, t domain.QueueThresholds) scoring.RuleResult {
	if t.AgeWarningSec == 0 {
		return ruleResult("QP-003", "Latência de processamento baixa", "warning", 5.0, false, true,
			"ℹ️ Thresholds não calculados ainda — regra não avaliada", "")
	}
	actual := fmt.Sprintf("%ds", snap.OldestUnackedAgeSec)
	passed := snap.OldestUnackedAgeSec < t.AgeWarningSec
	msg := fmt.Sprintf("✅ Latência dentro do threshold: %ds (warning: %ds)", snap.OldestUnackedAgeSec, t.AgeWarningSec)
	if !passed {
		msg = fmt.Sprintf("❌ Latência elevada: %ds (threshold warning: %ds)", snap.OldestUnackedAgeSec, t.AgeWarningSec)
	}
	return ruleResult("QP-003", "Latência de processamento baixa", "warning", 5.0, false, passed, msg, actual)
}
