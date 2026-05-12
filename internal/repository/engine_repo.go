package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/domain"
)

type EngineRepo struct{ db *pgxpool.Pool }

func NewEngineRepo(db *pgxpool.Pool) *EngineRepo { return &EngineRepo{db: db} }

func (r *EngineRepo) List(ctx context.Context) ([]domain.Engine, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, slug, name, COALESCE(description,''), enabled, created_at
		FROM titlis_config.scoring_engines
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Engine
	for rows.Next() {
		var e domain.Engine
		if err := rows.Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.Enabled, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EngineRepo) SlugForID(ctx context.Context, engineID int) (string, error) {
	var slug string
	err := r.db.QueryRow(ctx,
		`SELECT slug FROM titlis_config.scoring_engines WHERE id = $1`, engineID).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return slug, err
}

func (r *EngineRepo) GetBySlug(ctx context.Context, slug string) (*domain.Engine, error) {
	var e domain.Engine
	err := r.db.QueryRow(ctx, `
		SELECT id, slug, name, COALESCE(description,''), enabled, created_at
		FROM titlis_config.scoring_engines WHERE slug = $1`, slug).
		Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.Enabled, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &e, err
}

func (r *EngineRepo) Create(ctx context.Context, req domain.CreateEngineRequest) (*domain.Engine, error) {
	var e domain.Engine
	err := r.db.QueryRow(ctx, `
		INSERT INTO titlis_config.scoring_engines (slug, name, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO NOTHING
		RETURNING id, slug, name, COALESCE(description,''), enabled, created_at`,
		req.Slug, req.Name, req.Description).
		Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.Enabled, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrConflict
	}
	return &e, err
}

func (r *EngineRepo) SetEnabled(ctx context.Context, slug string, enabled bool) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE titlis_config.scoring_engines SET enabled = $1 WHERE slug = $2`,
		enabled, slug)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EngineRepo) ListRules(ctx context.Context, engineID int) ([]domain.Rule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, engine_id, rule_id, pillar, name, COALESCE(description,''), severity, enabled_by_default, created_at
		FROM titlis_config.engine_rules
		WHERE engine_id = $1
		ORDER BY pillar, rule_id`, engineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Rule
	for rows.Next() {
		var ru domain.Rule
		if err := rows.Scan(&ru.ID, &ru.EngineID, &ru.RuleID, &ru.Pillar, &ru.Name,
			&ru.Description, &ru.Severity, &ru.EnabledByDefault, &ru.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ru)
	}
	return out, rows.Err()
}

func (r *EngineRepo) CreateRule(ctx context.Context, engineID int, req domain.CreateRuleRequest) (*domain.Rule, error) {
	enabledByDefault := true
	if req.EnabledByDefault != nil {
		enabledByDefault = *req.EnabledByDefault
	}
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}

	var ru domain.Rule
	err := r.db.QueryRow(ctx, `
		INSERT INTO titlis_config.engine_rules
			(engine_id, rule_id, pillar, name, description, severity, enabled_by_default)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (engine_id, rule_id) DO NOTHING
		RETURNING id, engine_id, rule_id, pillar, name, COALESCE(description,''), severity, enabled_by_default, created_at`,
		engineID, req.RuleID, req.Pillar, req.Name, req.Description, severity, enabledByDefault).
		Scan(&ru.ID, &ru.EngineID, &ru.RuleID, &ru.Pillar, &ru.Name,
			&ru.Description, &ru.Severity, &ru.EnabledByDefault, &ru.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrConflict
	}
	return &ru, err
}
