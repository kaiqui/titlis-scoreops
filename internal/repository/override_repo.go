package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/domain"
)

type OverrideRepo struct{ db *pgxpool.Pool }

func NewOverrideRepo(db *pgxpool.Pool) *OverrideRepo { return &OverrideRepo{db: db} }

func (r *OverrideRepo) List(ctx context.Context, tenantID int, engineSlug, scope, cluster string) ([]domain.Override, error) {
	query := `
		SELECT o.rule_override_id, o.tenant_id, o.scoring_engine_id, o.rule_id, o.scope,
		       COALESCE(o.cluster_name,''), COALESCE(o.namespace,''), COALESCE(o.workload_uid,''),
		       o.enabled, COALESCE(o.reason,''), o.created_by, o.created_at
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.scoring_engine_id = o.scoring_engine_id
		WHERE o.tenant_id = $1 AND o.deleted_at IS NULL`

	args := []any{tenantID}
	n := 2

	if engineSlug != "" {
		query += ` AND e.slug = $` + itoa(n)
		args = append(args, engineSlug)
		n++
	}
	if scope != "" {
		query += ` AND o.scope = $` + itoa(n) + `::titlis_config.scope_type`
		args = append(args, scope)
		n++
	}
	if cluster != "" {
		query += ` AND o.cluster_name = $` + itoa(n)
		args = append(args, cluster)
		n++
	}
	_ = n
	query += ` ORDER BY o.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Override
	for rows.Next() {
		var o domain.Override
		if err := rows.Scan(&o.ID, &o.TenantID, &o.EngineID, &o.RuleID, &o.Scope,
			&o.ClusterName, &o.Namespace, &o.WorkloadUID,
			&o.Enabled, &o.Reason, &o.CreatedBy, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OverrideRepo) Upsert(ctx context.Context, tenantID int, req domain.CreateOverrideRequest) (*domain.Override, error) {
	var o domain.Override
	err := r.db.QueryRow(ctx, `
		INSERT INTO titlis_config.rule_overrides
			(tenant_id, scoring_engine_id, rule_id, scope, cluster_name, namespace, workload_uid, enabled, reason, created_by)
		VALUES ($1,$2,$3,$4::titlis_config.scope_type,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (tenant_id, scoring_engine_id, rule_id, scope,
		             COALESCE(cluster_name,''), COALESCE(namespace,''), COALESCE(workload_uid,''))
		DO UPDATE SET enabled = EXCLUDED.enabled, reason = EXCLUDED.reason,
		              created_by = EXCLUDED.created_by, created_at = NOW(), deleted_at = NULL
		RETURNING rule_override_id, tenant_id, scoring_engine_id, rule_id, scope,
		          COALESCE(cluster_name,''), COALESCE(namespace,''), COALESCE(workload_uid,''),
		          enabled, COALESCE(reason,''), created_by, created_at`,
		tenantID, req.EngineID, req.RuleID, string(req.Scope),
		nullStr(req.ClusterName), nullStr(req.Namespace), nullStr(req.WorkloadUID),
		req.Enabled, nullStr(req.Reason), req.CreatedBy).
		Scan(&o.ID, &o.TenantID, &o.EngineID, &o.RuleID, &o.Scope,
			&o.ClusterName, &o.Namespace, &o.WorkloadUID,
			&o.Enabled, &o.Reason, &o.CreatedBy, &o.CreatedAt)
	return &o, err
}

func (r *OverrideRepo) GetByID(ctx context.Context, tenantID int, id int64) (*domain.Override, error) {
	var o domain.Override
	err := r.db.QueryRow(ctx, `
		SELECT rule_override_id, tenant_id, scoring_engine_id, rule_id, scope,
		       COALESCE(cluster_name,''), COALESCE(namespace,''), COALESCE(workload_uid,''),
		       enabled, COALESCE(reason,''), created_by, created_at
		FROM titlis_config.rule_overrides
		WHERE rule_override_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID).
		Scan(&o.ID, &o.TenantID, &o.EngineID, &o.RuleID, &o.Scope,
			&o.ClusterName, &o.Namespace, &o.WorkloadUID,
			&o.Enabled, &o.Reason, &o.CreatedBy, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OverrideRepo) Delete(ctx context.Context, tenantID int, id int64) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE titlis_config.rule_overrides SET deleted_at = NOW()
		 WHERE rule_override_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ResolveClusterDisabled retorna rule_ids desativados no nível tenant ou cluster.
// Usado pelo operator para filtrar regras antes de avaliar qualquer workload do cluster.
func (r *OverrideRepo) ResolveClusterDisabled(ctx context.Context, tenantID int, engineSlug, clusterName string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.rule_id
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.scoring_engine_id = o.scoring_engine_id
		WHERE o.tenant_id = $1
		  AND e.slug = $2
		  AND o.enabled = FALSE
		  AND o.deleted_at IS NULL
		  AND (
		      (o.scope = 'tenant')
		   OR (o.scope = 'cluster' AND o.cluster_name = $3)
		  )`,
		tenantID, engineSlug, clusterName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

// ResolveWorkloadDisabled retorna rule_ids desativados no nível namespace ou workload.
func (r *OverrideRepo) ResolveWorkloadDisabled(ctx context.Context, tenantID int, engineSlug, clusterName, namespace, workloadUID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.rule_id
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.scoring_engine_id = o.scoring_engine_id
		WHERE o.tenant_id = $1
		  AND e.slug = $2
		  AND o.enabled = FALSE
		  AND o.deleted_at IS NULL
		  AND (
		      (o.scope = 'namespace' AND o.cluster_name = $3 AND o.namespace = $4)
		   OR (o.scope = 'workload'  AND o.cluster_name = $3 AND o.namespace = $4 AND o.workload_uid = $5)
		  )`,
		tenantID, engineSlug, clusterName, namespace, workloadUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

// Resolve avalia a hierarquia completa para uma regra específica de um workload.
func (r *OverrideRepo) Resolve(ctx context.Context, tenantID int, q domain.ResolveQuery) (*domain.ResolveResult, error) {
	// Busca o override de maior precedência (workload > namespace > cluster > tenant)
	var ruleID string
	var enabled bool
	var scope domain.ScopeType

	err := r.db.QueryRow(ctx, `
		SELECT o.rule_id, o.enabled, o.scope
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.scoring_engine_id = o.scoring_engine_id
		WHERE o.tenant_id = $1
		  AND e.slug = $2
		  AND o.rule_id = $3
		  AND o.deleted_at IS NULL
		  AND (
		      (o.scope = 'tenant')
		   OR (o.scope = 'cluster'    AND o.cluster_name = $4)
		   OR (o.scope = 'namespace'  AND o.cluster_name = $4 AND o.namespace = $5)
		   OR (o.scope = 'workload'   AND o.cluster_name = $4 AND o.namespace = $5 AND o.workload_uid = $6)
		  )
		ORDER BY CASE o.scope
		  WHEN 'workload'   THEN 1
		  WHEN 'namespace'  THEN 2
		  WHEN 'cluster'    THEN 3
		  WHEN 'tenant'     THEN 4
		END
		LIMIT 1`,
		tenantID, q.Engine, q.RuleID, q.ClusterName, q.Namespace, q.WorkloadUID).
		Scan(&ruleID, &enabled, &scope)

	if errors.Is(err, pgx.ErrNoRows) {
		// Sem override: comportamento padrão da regra
		return &domain.ResolveResult{RuleID: q.RuleID, Enabled: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.ResolveResult{RuleID: ruleID, Enabled: enabled, OverriddenBy: scope}, nil
}

func scanStrings(rows pgx.Rows) ([]string, error) {
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
