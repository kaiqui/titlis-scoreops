package domain

import "time"

type TagPolicy struct {
	ID        int64      `json:"id"`
	TenantID  int64      `json:"tenant_id"`
	Tag       string     `json:"tag"`
	RuleID    string     `json:"rule_id,omitempty"`
	Severity  string     `json:"severity,omitempty"`
	Action    string     `json:"action"`
	CreatedBy string     `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateTagPolicyRequest struct {
	Tag       string `json:"tag"`
	RuleID    string `json:"rule_id"`
	Severity  string `json:"severity"`
	Action    string `json:"action"`
	CreatedBy string `json:"created_by"`
}

func (r CreateTagPolicyRequest) Validate() error {
	if r.Tag == "" {
		return validationError("tag é obrigatória")
	}
	if (r.RuleID == "") == (r.Severity == "") {
		return validationError("informe rule_id OU severity, não ambos nem nenhum")
	}
	return nil
}
