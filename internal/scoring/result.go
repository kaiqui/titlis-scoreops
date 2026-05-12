package scoring

import "time"

type RuleResult struct {
	RuleID       string  `json:"rule_id"`
	RuleName     string  `json:"rule_name"`
	Passed       bool    `json:"passed"`
	Severity     string  `json:"severity"` // "critical" | "error" | "warning" | "info"
	Weight       float64 `json:"weight"`
	Message      string  `json:"message"`
	ActualValue  string  `json:"actual_value,omitempty"`
	IsRemediable bool    `json:"is_remediable"`
}

type PillarScore struct {
	Pillar        string  `json:"pillar"`
	Score         float64 `json:"score"`
	PassedChecks  int     `json:"passed_checks"`
	TotalChecks   int     `json:"total_checks"`
	WeightedScore float64 `json:"weighted_score"`
	Weight        float64 `json:"weight"`
}

type ScoreResult struct {
	WorkloadUID      string        `json:"workload_uid"`
	WorkloadName     string        `json:"workload_name"`
	Namespace        string        `json:"namespace"`
	Cluster          string        `json:"cluster"`
	TenantID         int64         `json:"tenant_id"`
	EngineSlug       string        `json:"engine_slug"`
	OverallScore     float64       `json:"overall_score"`
	ComplianceStatus string        `json:"compliance_status"` // "COMPLIANT" | "NON_COMPLIANT"
	CriticalIssues   int           `json:"critical_issues"`
	ErrorIssues      int           `json:"error_issues"`
	WarningIssues    int           `json:"warning_issues"`
	PassedChecks     int           `json:"passed_checks"`
	TotalChecks      int           `json:"total_checks"`
	RulesHash        string        `json:"rules_hash"`
	PillarScores     []PillarScore `json:"pillar_scores"`
	Findings         []RuleResult  `json:"findings"`
	CalculatedAt     time.Time     `json:"calculated_at"`
}
