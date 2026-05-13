package db

import (
	"context"
	"database/sql"
	"fmt"
)

// ConfigRepo implements domain.ConfigRepository with SQLite.
type ConfigRepo struct {
	store *SQLite
}

// NewConfigRepo creates a new ConfigRepo.
func NewConfigRepo(store *SQLite) *ConfigRepo {
	return &ConfigRepo{store: store}
}

func (r *ConfigRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.store.Db.QueryRowContext(ctx,
		`SELECT value FROM config WHERE key = ?`, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("config: %q not found", key)
	}
	if err != nil {
		return "", fmt.Errorf("config: get %q: %w", key, err)
	}
	return value, nil
}

func (r *ConfigRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("config: set %q: %w", key, err)
	}
	return nil
}
