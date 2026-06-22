package coverage

import (
	"math"
	"time"
)

// Engine evaluates a CoverageSnapshot against the expectation templates, producing a personalized
// scorecard + Trust Score. Pure and deterministic — no AI, no I/O.
type Engine struct {
	templates []ExpectationTemplate
}

func NewEngine(templates []ExpectationTemplate) *Engine {
	if templates == nil {
		templates = DefaultTemplates()
	}
	return &Engine{templates: templates}
}

func (e *Engine) Evaluate(snap CoverageSnapshot) CoverageResult {
	findings := make([]CoverageFinding, 0, len(e.templates))
	dims := map[string]*DimensionCoverage{}
	order := []string{}
	var weightTotal, weightPassed float64

	dim := func(pillar string) *DimensionCoverage {
		d, ok := dims[pillar]
		if !ok {
			d = &DimensionCoverage{Pillar: pillar}
			dims[pillar] = d
			order = append(order, pillar)
		}
		return d
	}

	for _, t := range e.templates {
		if !t.AppliesWhen(snap.Nature) {
			continue // not applicable to this service's nature → not emitted, no weight
		}
		d := dim(t.Pillar)

		// Applicable but the capability isn't measurable → N/A (never "missing"; OBS-002 pattern).
		if !snap.hasCapability(t.RequiresCapability) {
			findings = append(findings, CoverageFinding{
				Code: t.Code, Pillar: t.Pillar, Severity: t.Severity, Weight: t.Weight,
				IsRemediable: t.Remediable, Outcome: OutcomeNA, Message: t.NAMessage,
			})
			d.NA++
			continue
		}

		passed := t.Signal(snap.Found)
		outcome, msg := OutcomeFail, t.FailMessage
		if passed {
			outcome, msg = OutcomePass, t.PassMessage
		}
		findings = append(findings, CoverageFinding{
			Code: t.Code, Pillar: t.Pillar, Severity: t.Severity, Weight: t.Weight,
			IsRemediable: t.Remediable, Outcome: outcome, Message: msg,
		})
		d.Evaluable++
		weightTotal += t.Weight
		if passed {
			d.Passed++
			weightPassed += t.Weight
		}
	}

	dimensions := make([]DimensionCoverage, 0, len(order))
	overallMaturity := 0 // "elo mais fraco" entre dimensões avaliáveis
	for _, p := range order {
		d := dims[p]
		if d.Evaluable > 0 {
			d.Pct = round1(float64(d.Passed) / float64(d.Evaluable) * 100)
			d.MaturityLevel = maturityFromPct(d.Pct)
			if overallMaturity == 0 || d.MaturityLevel < overallMaturity {
				overallMaturity = d.MaturityLevel
			}
		} else {
			d.Pct = 100 // nothing evaluable in this dimension (all N/A) → not penalized
			d.MaturityLevel = 0
		}
		dimensions = append(dimensions, *d)
	}

	// Trust = weighted pass ratio over evaluable findings. K8s-ops templates are always evaluable,
	// so weightTotal > 0 in practice; the guard covers the all-N/A corner.
	trust := 100.0
	if weightTotal > 0 {
		trust = round1(weightPassed / weightTotal * 100)
	}

	return CoverageResult{
		WorkloadUID: snap.WorkloadUID,
		ServiceName: snap.ServiceName,
		Namespace:   snap.Namespace,
		Cluster:     snap.Cluster,
		TenantID:    snap.TenantID,
		EngineSlug:  "coverage",
		TrustScore:  trust,
		Maturity:    overallMaturity,
		Findings:    findings,
		Dimensions:  dimensions,
		EvaluatedAt: time.Now().UTC(),
	}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
