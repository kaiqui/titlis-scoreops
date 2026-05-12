package scoring

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultResilienceWeight  = 40.0
	defaultSecurityWeight    = 30.0
	defaultPerformanceWeight = 20.0
	defaultOperationalWeight = 10.0
)

// ContextResolver resolves which rules are active for a given workload and what pillar weights to use.
// It reads rule_overrides and pillar_weights from the DB, applying the full scope hierarchy.
type ContextResolver struct {
	db      *pgxpool.Pool
	pillars []PillarModule
}

func NewContextResolver(db *pgxpool.Pool, pillars []PillarModule) *ContextResolver {
	return &ContextResolver{db: db, pillars: pillars}
}

// ResolveActiveRules returns a map of ruleID → true/false based on DB overrides at all scopes,
// plus a SHA-256 hash of the effective disabled set (used for cache invalidation).
func (r *ContextResolver) ResolveActiveRules(
	ctx        context.Context,
	tenantID   int64,
	engineSlug string,
	snap       WorkloadSnapshot,
) (activeRules map[string]bool, hash string, err error) {
	allRuleIDs := r.collectRuleIDs()

	clusterDisabled, err := r.queryClusterDisabled(ctx, int(tenantID), engineSlug, snap.Cluster)
	if err != nil {
		return nil, "", fmt.Errorf("resolve cluster overrides: %w", err)
	}

	workloadDisabled, err := r.queryWorkloadDisabled(ctx, int(tenantID), engineSlug, snap.Cluster, snap.Namespace, snap.UID)
	if err != nil {
		return nil, "", fmt.Errorf("resolve workload overrides: %w", err)
	}

	allTags := append(snap.ClusterTags, snap.NamespaceTags...)
	tagDisabled, err := r.queryTagDisabled(ctx, tenantID, engineSlug, allTags)
	if err != nil {
		return nil, "", fmt.Errorf("resolve tag policies: %w", err)
	}

	disabled := make(map[string]bool, len(clusterDisabled)+len(workloadDisabled)+len(tagDisabled))
	for _, id := range clusterDisabled {
		disabled[id] = true
	}
	for _, id := range workloadDisabled {
		disabled[id] = true
	}
	for _, id := range tagDisabled {
		disabled[id] = true
	}

	active := make(map[string]bool, len(allRuleIDs))
	for _, id := range allRuleIDs {
		active[id] = !disabled[id]
	}

	disabledList := make([]string, 0, len(disabled))
	for id := range disabled {
		disabledList = append(disabledList, id)
	}
	sort.Strings(disabledList)
	sum := sha256.Sum256([]byte(fmt.Sprint(disabledList)))
	hash = fmt.Sprintf("%x", sum)

	return active, hash, nil
}

// ResolveWeights returns pillar weights for the tenant/engine, falling back to defaults for missing entries.
func (r *ContextResolver) ResolveWeights(
	ctx        context.Context,
	tenantID   int64,
	engineSlug string,
) (map[string]float64, error) {
	var engineID int
	err := r.db.QueryRow(ctx,
		`SELECT id FROM titlis_config.scoring_engines WHERE slug = $1`, engineSlug).
		Scan(&engineID)
	if err != nil {
		return defaultWeights(), nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT pillar, weight FROM titlis_config.pillar_weights WHERE tenant_id = $1 AND engine_id = $2`,
		tenantID, engineID)
	if err != nil {
		return defaultWeights(), nil
	}
	defer rows.Close()

	weights := defaultWeights()
	for rows.Next() {
		var pillar string
		var weight float64
		if err := rows.Scan(&pillar, &weight); err != nil {
			continue
		}
		weights[pillar] = weight
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return weights, nil
}

func (r *ContextResolver) queryClusterDisabled(
	ctx         context.Context,
	tenantID    int,
	engineSlug  string,
	clusterName string,
) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.rule_id
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.id = o.engine_id
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
	return scanStrings(rows)
}

func (r *ContextResolver) queryWorkloadDisabled(
	ctx         context.Context,
	tenantID    int,
	engineSlug  string,
	clusterName string,
	namespace   string,
	workloadUID string,
) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.rule_id
		FROM titlis_config.rule_overrides o
		JOIN titlis_config.scoring_engines e ON e.id = o.engine_id
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
	return scanStrings(rows)
}

// queryTagDisabled retorna rule_ids a desabilitar com base nas tag_rule_policies do tenant.
// Políticas por severity são expandidas para rule_ids via engine_rules no banco.
// Retorna nil sem erro quando tags está vazio (workloads sem tags não têm overhead extra).
func (r *ContextResolver) queryTagDisabled(
	ctx        context.Context,
	tenantID   int64,
	engineSlug string,
	tags       []string,
) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT er.rule_id
		FROM titlis_config.tag_rule_policies trp
		JOIN titlis_config.engine_rules er ON er.severity = trp.severity
		JOIN titlis_config.scoring_engines e ON e.id = er.engine_id AND e.slug = $3
		WHERE trp.tenant_id = $1
		  AND trp.tag = ANY($2)
		  AND trp.severity IS NOT NULL
		  AND trp.deleted_at IS NULL
		UNION
		SELECT rule_id
		FROM titlis_config.tag_rule_policies
		WHERE tenant_id = $1
		  AND tag = ANY($2)
		  AND rule_id IS NOT NULL
		  AND deleted_at IS NULL`,
		tenantID, tags, engineSlug)
	if err != nil {
		return nil, err
	}
	return scanStrings(rows)
}

func (r *ContextResolver) collectRuleIDs() []string {
	var ids []string
	seen := make(map[string]bool)
	for _, p := range r.pillars {
		for _, id := range p.RuleIDs() {
			if !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
	}
	return ids
}

func scanStrings(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
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

func defaultWeights() map[string]float64 {
	return map[string]float64{
		"resilience":  defaultResilienceWeight,
		"security":    defaultSecurityWeight,
		"performance": defaultPerformanceWeight,
		"operational": defaultOperationalWeight,
	}
}
