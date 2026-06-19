package scoring

import (
	"time"

	"github.com/titlis/scoreops/internal/domain"
)

// QueuePillarModule is the evaluation contract for queue scoring pillars.
type QueuePillarModule interface {
	Slug() string
	RuleIDs() []string
	Evaluate(snap domain.QueueSnapshot, active map[string]bool, thresholds domain.QueueThresholds, registry domain.LabelRegistry) []RuleResult
}

type QueueScoreResult struct {
	Provider         string        `json:"provider"`
	ExternalID       string        `json:"external_id"`
	TenantID         int64         `json:"tenant_id"`
	OverallScore     float64       `json:"overall_score"`
	ComplianceStatus string        `json:"compliance_status"`
	TotalChecks      int           `json:"total_checks"`
	PassedChecks     int           `json:"passed_checks"`
	CriticalIssues   int           `json:"critical_issues"`
	ErrorIssues      int           `json:"error_issues"`
	WarningIssues    int           `json:"warning_issues"`
	PillarScores     []PillarScore `json:"pillar_scores"`
	Findings         []RuleResult  `json:"findings"`
	CalculatedAt     time.Time     `json:"calculated_at"`
}

var defaultQueuePillarWeights = map[string]float64{
	"resilience":    35,
	"security":      25,
	"performance":   20,
	"operational":   10,
	"observability": 10,
}

type QueueScoreEngine struct {
	pillars []QueuePillarModule
}

func NewQueueScoreEngine() *QueueScoreEngine {
	return &QueueScoreEngine{}
}

func (e *QueueScoreEngine) RegisterPillar(p QueuePillarModule) {
	e.pillars = append(e.pillars, p)
}

func (e *QueueScoreEngine) Pillars() []QueuePillarModule { return e.pillars }

func (e *QueueScoreEngine) Evaluate(
	snap       domain.QueueSnapshot,
	active     map[string]bool,
	thresholds domain.QueueThresholds,
	registry   domain.LabelRegistry,
	weights    map[string]float64,
) QueueScoreResult {
	var (
		allFindings  []RuleResult
		pillarScores []PillarScore
		totalWeight  float64
		weightedSum  float64
	)

	for _, p := range e.pillars {
		results := p.Evaluate(snap, active, thresholds, registry)
		if len(results) == 0 {
			continue
		}

		pillarWeight := weights[p.Slug()]
		if pillarWeight == 0 {
			pillarWeight = defaultQueuePillarWeights[p.Slug()]
		}

		var passedW, totalW float64
		var passedCount int
		for _, r := range results {
			totalW += r.Weight
			if r.Passed {
				passedW += r.Weight
				passedCount++
			}
		}

		var pillarScore float64
		if totalW > 0 {
			pillarScore = (passedW / totalW) * 100
		}

		pillarScores = append(pillarScores, PillarScore{
			Pillar:        p.Slug(),
			Score:         pillarScore,
			PassedChecks:  passedCount,
			TotalChecks:   len(results),
			WeightedScore: pillarScore * pillarWeight,
			Weight:        pillarWeight,
		})

		totalWeight += pillarWeight
		weightedSum += pillarScore * pillarWeight
		allFindings = append(allFindings, results...)
	}

	var overallScore float64
	if totalWeight > 0 {
		overallScore = weightedSum / totalWeight
	}

	var criticals, errs, warnings, passed, total int
	for _, f := range allFindings {
		total++
		if f.Passed {
			passed++
		} else {
			switch f.Severity {
			case "critical":
				criticals++
			case "error":
				errs++
			case "warning":
				warnings++
			}
		}
	}

	compliance := "NON_COMPLIANT"
	if overallScore >= complianceThreshold {
		compliance = "COMPLIANT"
	}

	return QueueScoreResult{
		Provider:         snap.Provider,
		ExternalID:       snap.ExternalID,
		TenantID:         snap.TenantID,
		OverallScore:     overallScore,
		ComplianceStatus: compliance,
		TotalChecks:      total,
		PassedChecks:     passed,
		CriticalIssues:   criticals,
		ErrorIssues:      errs,
		WarningIssues:    warnings,
		PillarScores:     pillarScores,
		Findings:         allFindings,
		CalculatedAt:     time.Now(),
	}
}

// AllRulesActive builds an active map with every rule from the engine set to true.
func (e *QueueScoreEngine) AllRulesActive() map[string]bool {
	active := make(map[string]bool)
	for _, p := range e.pillars {
		for _, id := range p.RuleIDs() {
			active[id] = true
		}
	}
	return active
}
