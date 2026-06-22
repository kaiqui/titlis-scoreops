package coverage

import "time"

type Outcome string

const (
	OutcomePass Outcome = "pass"
	OutcomeFail Outcome = "fail"
	OutcomeNA   Outcome = "na" // applicable, but the required source is not connected
)

// CoverageFinding is one personalized scorecard item. Code is the STABLE template id (COV-*) so
// overrides/history/RAG/UI keep working; the personalization is which findings exist for this service.
type CoverageFinding struct {
	Code         string  `json:"code"`
	Pillar       string  `json:"pillar"`
	Severity     string  `json:"severity"`
	Weight       float64 `json:"weight"`
	IsRemediable bool    `json:"is_remediable"`
	Outcome      Outcome `json:"outcome"`
	Message      string  `json:"message"`
}

type DimensionCoverage struct {
	Pillar        string  `json:"pillar"`
	Evaluable     int     `json:"evaluable"` // applicable and not N/A
	Passed        int     `json:"passed"`
	NA            int     `json:"na"`
	Pct           float64 `json:"pct"`
	MaturityLevel int     `json:"maturity_level"` // 1–5 derivado do pct (0 = nada avaliável)
}

type CoverageResult struct {
	WorkloadUID string              `json:"workload_uid"`
	ServiceName string              `json:"service_name"`
	Namespace   string              `json:"namespace"`
	Cluster     string              `json:"cluster"`
	TenantID    int64               `json:"tenant_id"`
	EngineSlug  string              `json:"engine_slug"`
	TrustScore  float64             `json:"trust_score"` // 0–100 over evaluable (non-N/A) findings
	Maturity    int                 `json:"maturity"`    // 1–5, "elo mais fraco" entre as dimensões avaliáveis
	Findings    []CoverageFinding   `json:"findings"`
	Dimensions  []DimensionCoverage `json:"dimensions"`
	EvaluatedAt time.Time           `json:"evaluated_at"`
}

// maturityFromPct mapeia cobertura % → nível 1–5 (bandas).
func maturityFromPct(pct float64) int {
	switch {
	case pct >= 100:
		return 5
	case pct >= 80:
		return 4
	case pct >= 60:
		return 3
	case pct >= 40:
		return 2
	default:
		return 1
	}
}
