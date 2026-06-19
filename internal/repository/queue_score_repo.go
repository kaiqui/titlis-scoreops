package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/scoring"
)

type QueueScoreRepo struct {
	pool *pgxpool.Pool
}

func NewQueueScoreRepo(pool *pgxpool.Pool) *QueueScoreRepo {
	return &QueueScoreRepo{pool: pool}
}

func (r *QueueScoreRepo) Upsert(ctx context.Context, result scoring.QueueScoreResult) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO titlis_config.queue_scores (
			tenant_id, provider, external_id,
			overall_score, compliance_status,
			total_rules, passed_rules, failed_rules, critical_failures,
			error_count, warning_count, evaluated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (tenant_id, provider, external_id) DO UPDATE SET
			overall_score     = EXCLUDED.overall_score,
			compliance_status = EXCLUDED.compliance_status,
			total_rules       = EXCLUDED.total_rules,
			passed_rules      = EXCLUDED.passed_rules,
			failed_rules      = EXCLUDED.total_rules - EXCLUDED.passed_rules,
			critical_failures = EXCLUDED.critical_failures,
			error_count       = EXCLUDED.error_count,
			warning_count     = EXCLUDED.warning_count,
			evaluated_at      = EXCLUDED.evaluated_at
	`,
		result.TenantID, result.Provider, result.ExternalID,
		result.OverallScore, result.ComplianceStatus,
		result.TotalChecks, result.PassedChecks, result.TotalChecks-result.PassedChecks,
		result.CriticalIssues, result.ErrorIssues, result.WarningIssues, result.CalculatedAt,
	)
	return err
}
