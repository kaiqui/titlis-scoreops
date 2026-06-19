package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/titlis/scoreops/internal/domain"
)

type QueueSnapshotRepo struct {
	pool *pgxpool.Pool
}

func NewQueueSnapshotRepo(pool *pgxpool.Pool) *QueueSnapshotRepo {
	return &QueueSnapshotRepo{pool: pool}
}

func (r *QueueSnapshotRepo) Upsert(ctx context.Context, snap domain.QueueSnapshot) error {
	raw, err := json.Marshal(snap.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO titlis_config.queue_snapshots (
			tenant_id, provider, external_id, display_name, is_dlq,
			num_undelivered_messages, oldest_unacked_age_seconds,
			pull_message_count_rate, send_message_count_rate, ack_message_count_rate,
			dead_letter_message_count, has_dlq_configured, has_snapshot_policy,
			has_monitor_backlog, has_monitor_age, has_monitor_dlq,
			message_retention_sec, raw_metrics, collected_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19
		)
		ON CONFLICT (tenant_id, provider, external_id) DO UPDATE SET
			display_name               = EXCLUDED.display_name,
			is_dlq                     = EXCLUDED.is_dlq,
			num_undelivered_messages   = EXCLUDED.num_undelivered_messages,
			oldest_unacked_age_seconds = EXCLUDED.oldest_unacked_age_seconds,
			pull_message_count_rate    = EXCLUDED.pull_message_count_rate,
			send_message_count_rate    = EXCLUDED.send_message_count_rate,
			ack_message_count_rate     = EXCLUDED.ack_message_count_rate,
			dead_letter_message_count  = EXCLUDED.dead_letter_message_count,
			has_dlq_configured         = EXCLUDED.has_dlq_configured,
			has_snapshot_policy        = EXCLUDED.has_snapshot_policy,
			has_monitor_backlog        = EXCLUDED.has_monitor_backlog,
			has_monitor_age            = EXCLUDED.has_monitor_age,
			has_monitor_dlq            = EXCLUDED.has_monitor_dlq,
			message_retention_sec      = EXCLUDED.message_retention_sec,
			raw_metrics                = EXCLUDED.raw_metrics,
			collected_at               = EXCLUDED.collected_at
	`,
		snap.TenantID, snap.Provider, snap.ExternalID, snap.DisplayName, snap.IsDLQ,
		snap.NumUndeliveredMessages, snap.OldestUnackedAgeSec,
		snap.PullMessageCountRate, snap.SendMessageCountRate, snap.AckMessageCountRate,
		snap.DeadLetterMessageCount, snap.HasDLQConfigured, snap.HasSnapshotPolicy,
		snap.HasMonitorBacklog, snap.HasMonitorAge, snap.HasMonitorDLQ,
		snap.MessageRetentionDurationSec, raw, snap.CollectedAt,
	)
	return err
}
