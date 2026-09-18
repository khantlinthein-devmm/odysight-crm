package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetAll returns stored raw values keyed by setting key.
func (r *Repository) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k string
		var v json.RawMessage
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return out, nil
}

func (r *Repository) Upsert(ctx context.Context, key string, value json.RawMessage) error {
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2::jsonb, now())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, value); err != nil {
		return fmt.Errorf("upsert setting %s: %w", key, err)
	}
	return nil
}

// EnsureSeed inserts defaults for keys that have no row yet.
func (r *Repository) EnsureSeed(ctx context.Context, defaults map[string]string) error {
	for k, v := range defaults {
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO settings (key, value) VALUES ($1, $2::jsonb) ON CONFLICT (key) DO NOTHING`,
			k, v); err != nil {
			return fmt.Errorf("seed setting %s: %w", k, err)
		}
	}
	return nil
}
