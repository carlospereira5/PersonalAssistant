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
		`CREATE TABLE IF NOT EXISTS scheduled_routines (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			cron_expr   TEXT NOT NULL,
			prompt      TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'paused')),
			last_run_at TEXT,
			last_error  TEXT DEFAULT '',
			created_at  TEXT NOT NULL DEFAULT (datetime('now'))
		) STRICT;`,
		// ── Memoria semántica (FTS5) ─────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS memory_facts (
			rowid      INTEGER PRIMARY KEY AUTOINCREMENT,
			key        TEXT NOT NULL UNIQUE,
			value      TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			expires_at TEXT
		) STRICT;`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
			key, value,
			content='memory_facts',
			content_rowid='rowid'
		);`,
		`CREATE TRIGGER IF NOT EXISTS memory_facts_ai AFTER INSERT ON memory_facts BEGIN
			INSERT INTO memory_fts(rowid, key, value) VALUES (new.rowid, new.key, new.value);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS memory_facts_ad AFTER DELETE ON memory_facts BEGIN
			INSERT INTO memory_fts(memory_fts, rowid, key, value) VALUES('delete', old.rowid, old.key, old.value);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS memory_facts_au AFTER UPDATE ON memory_facts BEGIN
			INSERT INTO memory_fts(memory_fts, rowid, key, value) VALUES('delete', old.rowid, old.key, old.value);
			INSERT INTO memory_fts(rowid, key, value) VALUES (new.rowid, new.key, new.value);
		END;`,
		// ── Deudas ─────────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS debts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			total_amount INTEGER NOT NULL,
			state TEXT NOT NULL DEFAULT 'PENDING' CHECK(state IN ('PENDING','PARTIAL','PAID')),
			description TEXT DEFAULT ''
		) STRICT;`,
		`CREATE TABLE IF NOT EXISTS debts_payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			debt_id INTEGER NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
			amount_int INTEGER NOT NULL,
			notes TEXT DEFAULT '',
			paid_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
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
