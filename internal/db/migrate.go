package db

import (
	"context"
	"fmt"
)

// MigrateContext crea el esquema inicial de la base de datos.
func (s *SQLite) MigrateContext(ctx context.Context) error {
	s.Logger.Info("Ejecutando migraciones...")

	queries := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY,
			task_name TEXT NOT NULL,
			deadline TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'PENDING',
			description TEXT DEFAULT ''
		) STRICT;`,
		`CREATE TABLE IF NOT EXISTS task_reminders (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			remind_at  TEXT NOT NULL,
			sent       INTEGER NOT NULL DEFAULT 0
		) STRICT;`,
		`CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		) STRICT;`,
	}

	for _, query := range queries {
		if _, err := s.Db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("error ejecutando migración: %w", err)
		}
	}

	s.Logger.Info("Migraciones completadas con éxito")
	return nil
}
