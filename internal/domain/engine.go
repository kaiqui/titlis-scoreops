package domain

import "time"

type Engine struct {
	ID          int       `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type Rule struct {
	ID               int       `json:"id"`
	EngineID         int       `json:"engine_id"`
	RuleID           string    `json:"rule_id"`
	Pillar           string    `json:"pillar"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Severity         string    `json:"severity"`
	EnabledByDefault bool      `json:"enabled_by_default"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateEngineRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r CreateEngineRequest) Validate() error {
	if r.Slug == "" {
		return validationError("slug é obrigatório")
	}
	if r.Name == "" {
		return validationError("name é obrigatório")
	}
	return nil
}

type CreateRuleRequest struct {
	RuleID           string `json:"rule_id"`
	Pillar           string `json:"pillar"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Severity         string `json:"severity"`
	EnabledByDefault *bool  `json:"enabled_by_default"`
}

func (r CreateRuleRequest) Validate() error {
	if r.RuleID == "" {
		return validationError("rule_id é obrigatório")
	}
	if r.Pillar == "" {
		return validationError("pillar é obrigatório")
	}
	if r.Name == "" {
		return validationError("name é obrigatório")
	}
	return nil
}

type PatchEngineEnabledRequest struct {
	Enabled bool `json:"enabled"`
}
