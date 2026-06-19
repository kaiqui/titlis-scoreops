package pillar

import (
	"fmt"
	"strings"

	"github.com/titlis/scoreops/internal/domain"
	"github.com/titlis/scoreops/internal/scoring"
)

var requiredLabelKeys = []string{"env", "team", "service"}

type QueueSecurityPillar struct{}

func NewQueueSecurityPillar() *QueueSecurityPillar { return &QueueSecurityPillar{} }

func (p *QueueSecurityPillar) Slug() string { return "security" }

func (p *QueueSecurityPillar) RuleIDs() []string {
	return []string{"QS-001", "QS-002"}
}

func (p *QueueSecurityPillar) Evaluate(
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
		{"QS-001", func() scoring.RuleResult { return checkQS001(snap, registry) }},
		{"QS-002", func() scoring.RuleResult { return checkQS002(snap) }},
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

func checkQS001(snap domain.QueueSnapshot, registry domain.LabelRegistry) scoring.RuleResult {
	var missing, unregistered []string

	for _, key := range requiredLabelKeys {
		val, ok := snap.Labels[key]
		if !ok || val == "" {
			missing = append(missing, key)
			continue
		}
		if len(registry) > 0 && !registry.ContainsValue(key, val) {
			unregistered = append(unregistered, fmt.Sprintf("%s=%s", key, val))
		}
	}

	if len(missing) > 0 {
		return ruleResult("QS-001", "Labels com valores registrados", "warning", 7.0, true, false,
			fmt.Sprintf("❌ Labels obrigatórias ausentes: %s", strings.Join(missing, ", ")), "")
	}
	if len(unregistered) > 0 {
		return ruleResult("QS-001", "Labels com valores registrados", "warning", 7.0, true, false,
			fmt.Sprintf("❌ Valores não registrados no LabelRegistry: %s", strings.Join(unregistered, ", ")), "")
	}
	return ruleResult("QS-001", "Labels com valores registrados", "warning", 7.0, false, true,
		"✅ Labels env, team e service presentes com valores registrados", "")
}

func checkQS002(snap domain.QueueSnapshot) scoring.RuleResult {
	isPublic := snap.Labels["iam_public"] == "true"
	if isPublic {
		return ruleResult("QS-002", "Subscription não pública", "error", 10.0, false, false,
			"❌ Subscription exposta publicamente (iam_public:true detectado)", "public")
	}
	return ruleResult("QS-002", "Subscription não pública", "error", 10.0, false, true,
		"✅ Subscription não exposta publicamente", "")
}
