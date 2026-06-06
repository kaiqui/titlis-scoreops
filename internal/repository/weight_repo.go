package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/domain"
)

type WeightRepo struct{ db *pgxpool.Pool }

func NewWeightRepo(db *pgxpool.Pool) *WeightRepo { return &WeightRepo{db: db} }

func (r *WeightRepo) Get(ctx context.Context, tenantID, engineID int) ([]domain.PillarWeight, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tenant_id, scoring_engine_id, pillar, weight, updated_by
		FROM titlis_config.pillar_weights
		WHERE tenant_id = $1 AND scoring_engine_id = $2
		ORDER BY pillar`,
		tenantID, engineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PillarWeight
	for rows.Next() {
		var w domain.PillarWeight
		if err := rows.Scan(&w.TenantID, &w.EngineID, &w.Pillar, &w.Weight, &w.UpdatedBy); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WeightRepo) Set(ctx context.Context, tenantID int, req domain.SetWeightsRequest) ([]domain.PillarWeight, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	for pillar, weight := range req.Weights {
		_, err := tx.Exec(ctx, `
			INSERT INTO titlis_config.pillar_weights (tenant_id, scoring_engine_id, pillar, weight, updated_by)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (tenant_id, scoring_engine_id, pillar)
			DO UPDATE SET weight = EXCLUDED.weight, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
			tenantID, req.EngineID, pillar, weight, req.UpdatedBy)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.Get(ctx, tenantID, req.EngineID)
}
