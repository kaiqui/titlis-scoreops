package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/domain"
)

type TagPolicyRepo struct{ db *pgxpool.Pool }

func NewTagPolicyRepo(db *pgxpool.Pool) *TagPolicyRepo { return &TagPolicyRepo{db: db} }

func (r *TagPolicyRepo) List(ctx context.Context, tenantID int64) ([]domain.TagPolicy, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tag_rule_policie_id, tenant_id, tag, COALESCE(rule_id,''), COALESCE(severity,''),
		       action, COALESCE(created_by,''), created_at
		FROM titlis_config.tag_rule_policies
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.TagPolicy
	for rows.Next() {
		var p domain.TagPolicy
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Tag, &p.RuleID, &p.Severity,
			&p.Action, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *TagPolicyRepo) Create(ctx context.Context, tenantID int64, createdBy string, req domain.CreateTagPolicyRequest) (*domain.TagPolicy, error) {
	action := req.Action
	if action == "" {
		action = "disable"
	}
	var p domain.TagPolicy
	err := r.db.QueryRow(ctx, `
		INSERT INTO titlis_config.tag_rule_policies
			(tenant_id, tag, rule_id, severity, action, created_by)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5, NULLIF($6,''))
		ON CONFLICT DO NOTHING
		RETURNING tag_rule_policie_id, tenant_id, tag, COALESCE(rule_id,''), COALESCE(severity,''),
		          action, COALESCE(created_by,''), created_at`,
		tenantID, req.Tag, req.RuleID, req.Severity, action, createdBy).
		Scan(&p.ID, &p.TenantID, &p.Tag, &p.RuleID, &p.Severity,
			&p.Action, &p.CreatedBy, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrConflict
	}
	return &p, err
}

func (r *TagPolicyRepo) Delete(ctx context.Context, tenantID, id int64) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE titlis_config.tag_rule_policies
		 SET deleted_at = NOW()
		 WHERE tag_rule_policie_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
