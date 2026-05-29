package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Connect(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// Bootstrap: cria schema e tabela de tracking na primeira execução.
	// Requer DDL apenas uma vez; nas reinicializações seguintes é no-op.
	_, bootstrapErr := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS titlis_config;
		CREATE TABLE IF NOT EXISTS titlis_config.schema_migrations (
			version    VARCHAR(64) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if bootstrapErr != nil {
		// Usuário sem DDL: verifica se a tabela já existe antes de desistir.
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM titlis_config.schema_migrations`,
		).Scan(&n); err != nil {
			return fmt.Errorf("schema_migrations inacessível (inicialize o schema com um usuário privilegiado): %w", bootstrapErr)
		}
		slog.Debug("schema_migrations já existe, bootstrap ignorado", "motivo", bootstrapErr)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")

		var already bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM titlis_config.schema_migrations WHERE version = $1)`,
			version,
		).Scan(&already)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if already {
			slog.Debug("migration already applied, skipping", "version", version)
			continue
		}

		sql, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}

		if _, err := pool.Exec(ctx,
			`INSERT INTO titlis_config.schema_migrations (version) VALUES ($1)`,
			version,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		slog.Info("migration applied", "version", version)
	}
	return nil
}
