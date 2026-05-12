package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/scoring"
)

type ScoreHistoryEntry struct {
	OverallScore float64               `json:"overall_score"`
	PillarScores []scoring.PillarScore `json:"pillar_scores"`
	Findings     []scoring.RuleResult  `json:"findings"`
	TriggerType  string                `json:"trigger_type"`
	RulesHash    string                `json:"rules_hash"`
	CalculatedAt time.Time             `json:"calculated_at"`
}

type ScoreRepo struct {
	pool *pgxpool.Pool
}

func NewScoreRepo(pool *pgxpool.Pool) *ScoreRepo {
	return &ScoreRepo{pool: pool}
}

func (r *ScoreRepo) UpsertScore(ctx context.Context, result scoring.ScoreResult) error {
	pillarJSON, err := json.Marshal(result.PillarScores)
	if err != nil {
		return fmt.Errorf("marshal pillar_json: %w", err)
	}
	findingsJSON, err := json.Marshal(result.Findings)
	if err != nil {
		return fmt.Errorf("marshal findings_json: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO titlis_config.workload_scores
		    (tenant_id, engine_slug, workload_uid, cluster, namespace, workload_name,
		     overall_score, compliance_status, critical_issues, error_issues, warning_issues,
		     passed_checks, total_checks, pillar_json, findings_json, rules_hash, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (tenant_id, engine_slug, workload_uid) DO UPDATE SET
		    cluster           = EXCLUDED.cluster,
		    namespace         = EXCLUDED.namespace,
		    workload_name     = EXCLUDED.workload_name,
		    overall_score     = EXCLUDED.overall_score,
		    compliance_status = EXCLUDED.compliance_status,
		    critical_issues   = EXCLUDED.critical_issues,
		    error_issues      = EXCLUDED.error_issues,
		    warning_issues    = EXCLUDED.warning_issues,
		    passed_checks     = EXCLUDED.passed_checks,
		    total_checks      = EXCLUDED.total_checks,
		    pillar_json       = EXCLUDED.pillar_json,
		    findings_json     = EXCLUDED.findings_json,
		    rules_hash        = EXCLUDED.rules_hash,
		    calculated_at     = EXCLUDED.calculated_at
	`,
		result.TenantID, result.EngineSlug, result.WorkloadUID,
		result.Cluster, result.Namespace, result.WorkloadName,
		result.OverallScore, result.ComplianceStatus,
		result.CriticalIssues, result.ErrorIssues, result.WarningIssues,
		result.PassedChecks, result.TotalChecks,
		pillarJSON, findingsJSON, result.RulesHash, result.CalculatedAt,
	)
	return err
}

func (r *ScoreRepo) AppendHistory(ctx context.Context, result scoring.ScoreResult, triggerType string) error {
	pillarJSON, err := json.Marshal(result.PillarScores)
	if err != nil {
		return fmt.Errorf("marshal pillar_json: %w", err)
	}
	findingsJSON, err := json.Marshal(result.Findings)
	if err != nil {
		return fmt.Errorf("marshal findings_json: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO titlis_config.score_history
		    (tenant_id, engine_slug, workload_uid, cluster, namespace,
		     overall_score, pillar_json, findings_json, trigger_type, rules_hash, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		result.TenantID, result.EngineSlug, result.WorkloadUID,
		result.Cluster, result.Namespace,
		result.OverallScore, pillarJSON, findingsJSON,
		triggerType, result.RulesHash, result.CalculatedAt,
	)
	return err
}

func (r *ScoreRepo) ListScores(ctx context.Context, tenantID int64, engineSlug, cluster, namespace string, limit int) ([]scoring.ScoreResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT tenant_id, engine_slug, workload_uid, cluster, namespace, workload_name,
		       overall_score, compliance_status, critical_issues, error_issues, warning_issues,
		       passed_checks, total_checks, pillar_json, findings_json, rules_hash, calculated_at
		FROM titlis_config.workload_scores
		WHERE tenant_id = $1 AND engine_slug = $2`

	args := []any{tenantID, engineSlug}
	idx := 3

	if cluster != "" {
		query += fmt.Sprintf(" AND cluster = $%d", idx)
		args = append(args, cluster)
		idx++
	}
	if namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", idx)
		args = append(args, namespace)
		idx++
	}

	query += fmt.Sprintf(" ORDER BY calculated_at DESC LIMIT $%d", idx)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list scores: %w", err)
	}
	defer rows.Close()

	return scanScoreResults(rows)
}

func (r *ScoreRepo) GetScore(ctx context.Context, tenantID int64, engineSlug, workloadUID string) (*scoring.ScoreResult, []ScoreHistoryEntry, error) {
	result, err := r.fetchScore(ctx, tenantID, engineSlug, workloadUID)
	if err != nil || result == nil {
		return result, nil, err
	}

	histRows, err := r.pool.Query(ctx, `
		SELECT overall_score, pillar_json, findings_json, trigger_type, rules_hash, calculated_at
		FROM titlis_config.score_history
		WHERE tenant_id = $1 AND engine_slug = $2 AND workload_uid = $3
		ORDER BY calculated_at DESC
		LIMIT 10
	`, tenantID, engineSlug, workloadUID)
	if err != nil {
		return result, nil, fmt.Errorf("list history: %w", err)
	}
	defer histRows.Close()

	var history []ScoreHistoryEntry
	for histRows.Next() {
		var h ScoreHistoryEntry
		var pillarJSON, findingsJSON []byte
		if err := histRows.Scan(&h.OverallScore, &pillarJSON, &findingsJSON, &h.TriggerType, &h.RulesHash, &h.CalculatedAt); err != nil {
			return result, nil, err
		}
		if err := json.Unmarshal(pillarJSON, &h.PillarScores); err != nil {
			return result, nil, err
		}
		if err := json.Unmarshal(findingsJSON, &h.Findings); err != nil {
			return result, nil, err
		}
		history = append(history, h)
	}
	if err := histRows.Err(); err != nil {
		return result, nil, err
	}

	return result, history, nil
}

func (r *ScoreRepo) fetchScore(ctx context.Context, tenantID int64, engineSlug, workloadUID string) (*scoring.ScoreResult, error) {
	var res scoring.ScoreResult
	var pillarJSON, findingsJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, engine_slug, workload_uid, cluster, namespace, workload_name,
		       overall_score, compliance_status, critical_issues, error_issues, warning_issues,
		       passed_checks, total_checks, pillar_json, findings_json, rules_hash, calculated_at
		FROM titlis_config.workload_scores
		WHERE tenant_id = $1 AND engine_slug = $2 AND workload_uid = $3
	`, tenantID, engineSlug, workloadUID).Scan(
		&res.TenantID, &res.EngineSlug, &res.WorkloadUID,
		&res.Cluster, &res.Namespace, &res.WorkloadName,
		&res.OverallScore, &res.ComplianceStatus,
		&res.CriticalIssues, &res.ErrorIssues, &res.WarningIssues,
		&res.PassedChecks, &res.TotalChecks,
		&pillarJSON, &findingsJSON, &res.RulesHash, &res.CalculatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch score: %w", err)
	}
	if err := json.Unmarshal(pillarJSON, &res.PillarScores); err != nil {
		return nil, fmt.Errorf("unmarshal pillar_json: %w", err)
	}
	if err := json.Unmarshal(findingsJSON, &res.Findings); err != nil {
		return nil, fmt.Errorf("unmarshal findings_json: %w", err)
	}
	return &res, nil
}

func scanScoreResults(rows pgx.Rows) ([]scoring.ScoreResult, error) {
	var results []scoring.ScoreResult
	for rows.Next() {
		var r scoring.ScoreResult
		var pillarJSON, findingsJSON []byte
		err := rows.Scan(
			&r.TenantID, &r.EngineSlug, &r.WorkloadUID,
			&r.Cluster, &r.Namespace, &r.WorkloadName,
			&r.OverallScore, &r.ComplianceStatus,
			&r.CriticalIssues, &r.ErrorIssues, &r.WarningIssues,
			&r.PassedChecks, &r.TotalChecks,
			&pillarJSON, &findingsJSON, &r.RulesHash, &r.CalculatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan score row: %w", err)
		}
		if err := json.Unmarshal(pillarJSON, &r.PillarScores); err != nil {
			return nil, fmt.Errorf("unmarshal pillar_json: %w", err)
		}
		if err := json.Unmarshal(findingsJSON, &r.Findings); err != nil {
			return nil, fmt.Errorf("unmarshal findings_json: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
