package pillar

import (
	"fmt"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

const dlqSaturationThreshold = 10

type QueueResiliencePillar struct{}

func NewQueueResiliencePillar() *QueueResiliencePillar { return &QueueResiliencePillar{} }

func (p *QueueResiliencePillar) Slug() string { return "resilience" }

func (p *QueueResiliencePillar) RuleIDs() []string {
	return []string{"QR-001", "QR-002", "QR-003", "QR-004", "QR-005"}
}

func (p *QueueResiliencePillar) Evaluate(
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
		{"QR-001", func() scoring.RuleResult { return checkQR001(snap, thresholds) }},
		{"QR-002", func() scoring.RuleResult { return checkQR002(snap, thresholds) }},
		{"QR-003", func() scoring.RuleResult { return checkQR003(snap) }},
		{"QR-004", func() scoring.RuleResult { return checkQR004(snap) }},
		{"QR-005", func() scoring.RuleResult { return checkQR005(snap) }},
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

func checkQR001(snap domain.QueueSnapshot, t domain.QueueThresholds) scoring.RuleResult {
	actual := fmt.Sprintf("%d", snap.NumUndeliveredMessages)
	if t.BacklogCritical > 0 && snap.NumUndeliveredMessages >= t.BacklogCritical {
		return ruleResult("QR-001", "Backlog dentro do threshold", "critical", 10.0, false, false,
			fmt.Sprintf("❌ Backlog crítico: %d mensagens (threshold: %d)", snap.NumUndeliveredMessages, t.BacklogCritical),
			actual)
	}
	if t.BacklogWarning > 0 && snap.NumUndeliveredMessages >= t.BacklogWarning {
		return ruleResult("QR-001", "Backlog dentro do threshold", "error", 10.0, false, false,
			fmt.Sprintf("⚠️ Backlog em alerta: %d mensagens (threshold warning: %d)", snap.NumUndeliveredMessages, t.BacklogWarning),
			actual)
	}
	msg := fmt.Sprintf("✅ Backlog saudável: %d mensagens", snap.NumUndeliveredMessages)
	if t.BacklogWarning > 0 {
		msg = fmt.Sprintf("✅ Backlog saudável: %d mensagens (threshold warning: %d)", snap.NumUndeliveredMessages, t.BacklogWarning)
	}
	return ruleResult("QR-001", "Backlog dentro do threshold", "error", 10.0, false, true, msg, actual)
}

func checkQR002(snap domain.QueueSnapshot, t domain.QueueThresholds) scoring.RuleResult {
	actual := fmt.Sprintf("%ds", snap.OldestUnackedAgeSec)
	if t.AgeCriticalSec > 0 && snap.OldestUnackedAgeSec >= t.AgeCriticalSec {
		return ruleResult("QR-002", "Mensagem mais antiga dentro do threshold", "error", 10.0, false, false,
			fmt.Sprintf("❌ Mensagem crítica: %ds de atraso (threshold crítico: %ds)", snap.OldestUnackedAgeSec, t.AgeCriticalSec),
			actual)
	}
	if t.AgeWarningSec > 0 && snap.OldestUnackedAgeSec >= t.AgeWarningSec {
		return ruleResult("QR-002", "Mensagem mais antiga dentro do threshold", "warning", 10.0, false, false,
			fmt.Sprintf("⚠️ Mensagem em alerta: %ds de atraso (threshold warning: %ds)", snap.OldestUnackedAgeSec, t.AgeWarningSec),
			actual)
	}
	msg := fmt.Sprintf("✅ Age saudável: %ds", snap.OldestUnackedAgeSec)
	if t.AgeWarningSec > 0 {
		msg = fmt.Sprintf("✅ Age saudável: %ds (threshold warning: %ds)", snap.OldestUnackedAgeSec, t.AgeWarningSec)
	}
	return ruleResult("QR-002", "Mensagem mais antiga dentro do threshold", "error", 10.0, false, true, msg, actual)
}

func checkQR003(snap domain.QueueSnapshot) scoring.RuleResult {
	if snap.IsDLQ {
		return ruleResult("QR-003", "DLQ configurada", "warning", 8.0, false, true,
			"✅ Subscription é DLQ — regra não aplicável", "")
	}
	passed := snap.HasDLQConfigured
	msg := "✅ DLQ configurada para esta subscription"
	if !passed {
		msg = "❌ Nenhuma DLQ configurada para esta subscription"
	}
	return ruleResult("QR-003", "DLQ configurada", "warning", 8.0, true, passed, msg, "")
}

func checkQR004(snap domain.QueueSnapshot) scoring.RuleResult {
	if !snap.IsDLQ {
		return ruleResult("QR-004", "DLQ não saturada", "critical", 10.0, false, true,
			"✅ Não é uma DLQ — regra não aplicável", "")
	}
	actual := fmt.Sprintf("%d", snap.NumUndeliveredMessages)
	passed := snap.NumUndeliveredMessages < dlqSaturationThreshold
	msg := fmt.Sprintf("✅ DLQ com %d mensagens", snap.NumUndeliveredMessages)
	if !passed {
		msg = fmt.Sprintf("❌ DLQ saturada: %d mensagens (threshold: %d)", snap.NumUndeliveredMessages, dlqSaturationThreshold)
	}
	return ruleResult("QR-004", "DLQ não saturada", "critical", 10.0, false, passed, msg, actual)
}

func checkQR005(snap domain.QueueSnapshot) scoring.RuleResult {
	const sevenDaysSec = 604800
	if snap.MessageRetentionDurationSec == 0 {
		return ruleResult("QR-005", "Retenção de mensagem ≥ 7d", "info", 3.0, false, true,
			"ℹ️ Retenção não reportada pelo coletor", "")
	}
	actual := fmt.Sprintf("%ds", snap.MessageRetentionDurationSec)
	passed := snap.MessageRetentionDurationSec >= sevenDaysSec
	msg := fmt.Sprintf("✅ Retenção configurada: %ds", snap.MessageRetentionDurationSec)
	if !passed {
		msg = fmt.Sprintf("❌ Retenção insuficiente: %ds (mínimo: %ds)", snap.MessageRetentionDurationSec, sevenDaysSec)
	}
	return ruleResult("QR-005", "Retenção de mensagem ≥ 7d", "info", 3.0, false, passed, msg, actual)
}
