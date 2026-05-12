package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID         int64           `json:"id"`
	TenantID   int             `json:"tenant_id"`
	Actor      string          `json:"actor"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AuditRepo struct{ db *pgxpool.Pool }

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Append(ctx context.Context, tenantID int, actor, action, entityType, entityID string, before, after any) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)

	var beforeArg any
	if before != nil {
		beforeArg = beforeJSON
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO titlis_config.config_audit_log
			(tenant_id, actor, action, entity_type, entity_id, before_json, after_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		tenantID, actor, action, entityType, entityID, beforeArg, afterJSON)
	return err
}

func (r *AuditRepo) List(ctx context.Context, tenantID, limit int, before int64) ([]AuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var rows pgx.Rows
	var err error

	query := `
		SELECT id, tenant_id, actor, action, entity_type, entity_id,
		       before_json, after_json, created_at
		FROM titlis_config.config_audit_log
		WHERE tenant_id = $1`

	if before > 0 {
		query += ` AND id < $3 ORDER BY created_at DESC, id DESC LIMIT $2`
		rows, err = r.db.Query(ctx, query, tenantID, limit, before)
	} else {
		query += ` ORDER BY created_at DESC, id DESC LIMIT $2`
		rows, err = r.db.Query(ctx, query, tenantID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Actor, &e.Action, &e.EntityType, &e.EntityID,
			&e.Before, &e.After, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
