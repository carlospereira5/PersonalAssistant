// Package db provides SQLite persistence for the personal assistant.
package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	_ "modernc.org/sqlite"
)

// SQLite wraps the database connection.
type SQLite struct {
	Db     *sql.DB
	Logger *log.Logger
}

// NewDB opens a SQLite connection with optimized pragmas.
func NewDB(path string, logger *log.Logger) (*SQLite, error) {
	dsn := path
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	dsn += "_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	logger.Info("Abriendo base de datos", "path", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al conectar con la base de datos: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &SQLite{Db: db, Logger: logger}, nil
}

// Close cierra la conexión de forma segura.
func (s *SQLite) Close() error {
	s.Logger.Info("Cerrando base de datos")
	return s.Db.Close()
}
