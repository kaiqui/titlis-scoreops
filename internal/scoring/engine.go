package scoring

import "time"

const complianceThreshold = 80.0

var defaultPillarWeights = map[string]float64{
	"resilience":  40,
	"security":    30,
	"performance": 20,
	"operational": 10,
}

// PillarModule is the evaluation contract for a scoring pillar.
// Implementations live in internal/pillar/*.go and import this package for the shared types.
type PillarModule interface {
	Slug() string
	RuleIDs() []string
	// Evaluate returns results only for rules present in activeRules with value true.
	Evaluate(snap WorkloadSnapshot, activeRules map[string]bool) []RuleResult
}

type ScoreEngine struct {
	pillars []PillarModule
	dag     *DAG
}

func NewScoreEngine() *ScoreEngine {
	return &ScoreEngine{dag: NewDAG()}
}

func (e *ScoreEngine) RegisterPillar(p PillarModule) {
	e.pillars = append(e.pillars, p)
	e.dag.AddNode(p.Slug())
}

func (e *ScoreEngine) Pillars() []PillarModule { return e.pillars }

// Evaluate computes a full ScoreResult for the given snapshot.
// active: ruleID → true if the rule should be evaluated.
// weights: pillarSlug → pillar weight; missing pillars fall back to defaultPillarWeights.
func (e *ScoreEngine) Evaluate(
	snap    WorkloadSnapshot,
	active  map[string]bool,
	weights map[string]float64,
) ScoreResult {
	order, err := e.dag.TopologicalSort()
	if err != nil {
		order = make([]string, len(e.pillars))
		for i, p := range e.pillars {
			order[i] = p.Slug()
		}
	}

	pillarMap := make(map[string]PillarModule, len(e.pillars))
	for _, p := range e.pillars {
		pillarMap[p.Slug()] = p
	}

	var (
		allFindings  []RuleResult
		pillarScores []PillarScore
		totalWeight  float64
		weightedSum  float64
	)

	for _, slug := range order {
		p, ok := pillarMap[slug]
		if !ok {
			continue
		}

		results := p.Evaluate(snap, active)

		pillarWeight := weights[slug]
		if pillarWeight == 0 {
			pillarWeight = defaultPillarWeights[slug]
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
			Pillar:        slug,
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

	return ScoreResult{
		WorkloadUID:      snap.UID,
		WorkloadName:     snap.Name,
		Namespace:        snap.Namespace,
		Cluster:          snap.Cluster,
		TenantID:         snap.TenantID,
		EngineSlug:       snap.EngineSlug,
		OverallScore:     overallScore,
		ComplianceStatus: compliance,
		CriticalIssues:   criticals,
		ErrorIssues:      errs,
		WarningIssues:    warnings,
		PassedChecks:     passed,
		TotalChecks:      total,
		PillarScores:     pillarScores,
		Findings:         allFindings,
		CalculatedAt:     time.Now(),
	}
}
