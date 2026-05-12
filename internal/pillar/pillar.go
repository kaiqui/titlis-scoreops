// Package pillar contains PillarModule implementations for the scoring engine.
// Each file implements the scoring.PillarModule interface for one pillar.
// All types shared between pillar and engine (WorkloadSnapshot, RuleResult, PillarModule)
// live in the scoring package to avoid circular imports.
package pillar

import "github.com/titlis/scoreops/internal/scoring"

func ruleResult(id, name, severity string, weight float64, remediable, passed bool, msg, actual string) scoring.RuleResult {
	return scoring.RuleResult{
		RuleID:       id,
		RuleName:     name,
		Severity:     severity,
		Weight:       weight,
		IsRemediable: remediable,
		Passed:       passed,
		Message:      msg,
		ActualValue:  actual,
	}
}
