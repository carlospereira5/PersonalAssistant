// Package memory — FactStore FTS5 para memoria semántica persistente.
//
// Los facts son pares key-value con búsqueda full-text via SQLite FTS5.
// Soportan TTL opcional para expiración automática.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/carlospereira5/PersonalAssistant/agent"
)

// Fact representa un hecho persistente en la memoria semántica.
type Fact struct {
	Key       string
	Value     string
	CreatedAt time.Time
	ExpiresAt *time.Time // nil = no expira
}

// FactStore gestiona la persistencia y búsqueda de facts via SQLite FTS5.
type FactStore struct {
	db agent.PortDB
}

// NewFactStore crea un FactStore.
func NewFactStore(db agent.PortDB) *FactStore {
	return &FactStore{db: db}
}

// SaveFact guarda o actualiza un fact. Si ya existe una fact con la misma key,
// se reemplaza. ttlSeconds <= 0 significa sin expiración.
func (s *FactStore) SaveFact(ctx context.Context, key, value string, ttlSeconds int) error {
	var expiresAt *string
	if ttlSeconds > 0 {
		exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second).Format(time.RFC3339)
		expiresAt = &exp
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO memory_facts (key, value, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			expires_at = excluded.expires_at,
			created_at = datetime('now')
	`, key, value, expiresAt)
	if err != nil {
		return fmt.Errorf("save_fact: %w", err)
	}
	return nil
}

// GetFact recupera un fact por su key. Retorna nil si no existe o expiró.
func (s *FactStore) GetFact(ctx context.Context, key string) (*Fact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT key, value, created_at, expires_at
		FROM memory_facts
		WHERE key = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, key)

	var f Fact
	var createdAt, expiresAt sql.NullString
	if err := row.Scan(&f.Key, &f.Value, &createdAt, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get_fact: %w", err)
	}

	if createdAt.Valid {
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if expiresAt.Valid {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err == nil {
			f.ExpiresAt = &t
		}
	}
	return &f, nil
}

// SearchFacts busca facts cuyo key o value coincida con la query.
// Usa FTS5 MATCH para búsqueda full-text. Soporta términos parciales.
func (s *FactStore) SearchFacts(ctx context.Context, query string) ([]Fact, error) {
	// FTS5 requiere que los términos de búsqueda estén entre comillas
	// para evitar errores de sintaxis con palabras reservadas.
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.key, f.value, f.created_at, f.expires_at
		FROM memory_facts f
		JOIN memory_fts ON f.rowid = memory_fts.rowid
		WHERE memory_fts MATCH ? AND (f.expires_at IS NULL OR f.expires_at > datetime('now'))
		ORDER BY rank
		LIMIT 20
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search_facts: %w", err)
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var createdAt, expiresAt sql.NullString
		if err := rows.Scan(&f.Key, &f.Value, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("search_facts scan: %w", err)
		}
		if createdAt.Valid {
			f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}
		if expiresAt.Valid {
			t, err := time.Parse(time.RFC3339, expiresAt.String)
			if err == nil {
				f.ExpiresAt = &t
			}
		}
		facts = append(facts, f)
	}
	if facts == nil {
		facts = []Fact{} // nunca retornar nil — el LLM espera un array vacío
	}
	return facts, rows.Err()
}

// GetAllFacts retorna todos los facts activos (no expirados).
// Útil para inyectar contexto al LLM en la construcción del prompt.
func (s *FactStore) GetAllFacts(ctx context.Context) ([]Fact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, value, created_at, expires_at
		FROM memory_facts
		WHERE expires_at IS NULL OR expires_at > datetime('now')
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("get_all_facts: %w", err)
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var createdAt, expiresAt sql.NullString
		if err := rows.Scan(&f.Key, &f.Value, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("get_all_facts scan: %w", err)
		}
		if createdAt.Valid {
			f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}
		if expiresAt.Valid {
			t, err := time.Parse(time.RFC3339, expiresAt.String)
			if err == nil {
				f.ExpiresAt = &t
			}
		}
		facts = append(facts, f)
	}
	if facts == nil {
		facts = []Fact{}
	}
	return facts, rows.Err()
}

// DeleteFact elimina un fact por su key.
func (s *FactStore) DeleteFact(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memory_facts WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete_fact: %w", err)
	}
	return nil
}
