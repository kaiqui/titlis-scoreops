package domain

import "time"

type ScopeType string

const (
	ScopeTenant    ScopeType = "tenant"
	ScopeCluster   ScopeType = "cluster"
	ScopeNamespace ScopeType = "namespace"
	ScopeWorkload  ScopeType = "workload"
)

func (s ScopeType) Valid() bool {
	switch s {
	case ScopeTenant, ScopeCluster, ScopeNamespace, ScopeWorkload:
		return true
	}
	return false
}

type Override struct {
	ID          int64     `json:"id"`
	TenantID    int       `json:"tenant_id"`
	EngineID    int       `json:"engine_id"`
	RuleID      string    `json:"rule_id"`
	Scope       ScopeType `json:"scope"`
	ClusterName string    `json:"cluster_name,omitempty"`
	Namespace   string    `json:"namespace,omitempty"`
	WorkloadUID string    `json:"workload_uid,omitempty"`
	Enabled     bool      `json:"enabled"`
	Reason      string    `json:"reason,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateOverrideRequest struct {
	EngineID    int       `json:"engine_id"`
	RuleID      string    `json:"rule_id"`
	Scope       ScopeType `json:"scope"`
	ClusterName string    `json:"cluster_name"`
	Namespace   string    `json:"namespace"`
	WorkloadUID string    `json:"workload_uid"`
	Enabled     bool      `json:"enabled"`
	Reason      string    `json:"reason"`
	CreatedBy   string    `json:"created_by"`
}

func (r CreateOverrideRequest) Validate() error {
	if r.EngineID == 0 {
		return validationError("engine_id é obrigatório")
	}
	if r.RuleID == "" {
		return validationError("rule_id é obrigatório")
	}
	if !r.Scope.Valid() {
		return validationError("scope inválido: use tenant, cluster, namespace ou workload")
	}
	if r.CreatedBy == "" {
		return validationError("created_by é obrigatório")
	}

	switch r.Scope {
	case ScopeCluster:
		if r.ClusterName == "" {
			return validationError("cluster_name obrigatório para scope=cluster")
		}
	case ScopeNamespace:
		if r.ClusterName == "" || r.Namespace == "" {
			return validationError("cluster_name e namespace obrigatórios para scope=namespace")
		}
	case ScopeWorkload:
		if r.ClusterName == "" || r.Namespace == "" || r.WorkloadUID == "" {
			return validationError("cluster_name, namespace e workload_uid obrigatórios para scope=workload")
		}
	}
	return nil
}

type ResolveQuery struct {
	Engine      string
	RuleID      string
	ClusterName string
	Namespace   string
	WorkloadUID string
}

type ResolveResult struct {
	RuleID       string    `json:"rule_id"`
	Enabled      bool      `json:"enabled"`
	OverriddenBy ScopeType `json:"overridden_by,omitempty"`
}
