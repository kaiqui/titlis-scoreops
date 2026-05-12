package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/scoring"
)

type SnapshotRepo struct {
	pool *pgxpool.Pool
}

func NewSnapshotRepo(pool *pgxpool.Pool) *SnapshotRepo {
	return &SnapshotRepo{pool: pool}
}

func (r *SnapshotRepo) UpsertSnapshot(ctx context.Context, snap scoring.WorkloadSnapshot, rulesHash string) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO titlis_config.workload_snapshots
		    (tenant_id, engine_slug, workload_uid, cluster, namespace, workload_name, metrics_json, rules_hash, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_id, engine_slug, workload_uid) DO UPDATE SET
		    cluster       = EXCLUDED.cluster,
		    namespace     = EXCLUDED.namespace,
		    workload_name = EXCLUDED.workload_name,
		    metrics_json  = EXCLUDED.metrics_json,
		    rules_hash    = EXCLUDED.rules_hash,
		    received_at   = NOW()
	`, snap.TenantID, snap.EngineSlug, snap.UID, snap.Cluster, snap.Namespace, snap.Name, data, rulesHash)
	return err
}

func (r *SnapshotRepo) GetSnapshot(ctx context.Context, tenantID int64, engineSlug, workloadUID string) (*scoring.WorkloadSnapshot, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT metrics_json FROM titlis_config.workload_snapshots
		WHERE tenant_id = $1 AND engine_slug = $2 AND workload_uid = $3
	`, tenantID, engineSlug, workloadUID)

	var data []byte
	if err := row.Scan(&data); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan snapshot: %w", err)
	}

	var snap scoring.WorkloadSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return &snap, nil
}

// ListUIDsInScope returns workload UIDs affected by a change at the given scope level.
// scope: "tenant" → all, "cluster" → by cluster, "namespace" → by cluster+namespace, "workload" → by workloadUID.
func (r *SnapshotRepo) ListUIDsInScope(ctx context.Context, tenantID int64, engineSlug, scope, cluster, namespace, workloadUID string) ([]string, error) {
	var query string
	var args []any

	switch scope {
	case "workload":
		query = `SELECT workload_uid FROM titlis_config.workload_snapshots
		          WHERE tenant_id = $1 AND engine_slug = $2 AND workload_uid = $3`
		args = []any{tenantID, engineSlug, workloadUID}
	case "namespace":
		query = `SELECT workload_uid FROM titlis_config.workload_snapshots
		          WHERE tenant_id = $1 AND engine_slug = $2 AND cluster = $3 AND namespace = $4`
		args = []any{tenantID, engineSlug, cluster, namespace}
	case "cluster":
		query = `SELECT workload_uid FROM titlis_config.workload_snapshots
		          WHERE tenant_id = $1 AND engine_slug = $2 AND cluster = $3`
		args = []any{tenantID, engineSlug, cluster}
	default: // "tenant" or unknown
		query = `SELECT workload_uid FROM titlis_config.workload_snapshots
		          WHERE tenant_id = $1 AND engine_slug = $2`
		args = []any{tenantID, engineSlug}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list uids in scope: %w", err)
	}
	defer rows.Close()

	return collectUIDs(rows)
}

func (r *SnapshotRepo) ListAllUIDs(ctx context.Context, tenantID int64, engineSlug string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT workload_uid FROM titlis_config.workload_snapshots
		WHERE tenant_id = $1 AND engine_slug = $2
	`, tenantID, engineSlug)
	if err != nil {
		return nil, fmt.Errorf("list all uids: %w", err)
	}
	defer rows.Close()

	return collectUIDs(rows)
}

func collectUIDs(rows pgx.Rows) ([]string, error) {
	var uids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, rows.Err()
}
