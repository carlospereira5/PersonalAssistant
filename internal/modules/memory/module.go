// Package memory — módulo Gen2 DataReader + DataWriter para memoria semántica.
//
// Expone tools para que el LLM guarde y recupere hechos (facts) de forma
// persistente entre sesiones, usando SQLite FTS5 para búsqueda full-text.
package memory

import (
	"context"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	memstore "github.com/carlospereira5/PersonalAssistant/internal/infrastructure/memory"
)

// Module implementa DataReader + DataWriter para memoria semántica.
type Module struct {
	store  *memstore.FactStore
	logger *charm.Logger
}

// New crea un módulo de memoria.
// Requiere Init() con un PortDeps que tenga DB configurada.
func New() *Module { return &Module{} }

func (m *Module) Name() string     { return "memory" }
func (m *Module) Schema() []string { return nil } // Schema gestionado en db/migrate.go

func (m *Module) Init(deps agent.PortDeps) error {
	m.logger = deps.Logger
	m.store = memstore.NewFactStore(deps.DB)
	return nil
}

// PromptSection informa al LLM que tiene memoria persistente disponible.
func (m *Module) PromptSection(ctx context.Context, _ string) string {
	return "\n## MEMORIA PERSISTENTE\n" +
		"Podés recordar información entre sesiones usando:\n" +
		"- save_fact(key, value, [ttl]): guardar un hecho\n" +
		"- get_fact(key): recuperar un hecho\n" +
		"- search_memory(query): buscar en todos los hechos guardados\n" +
		"Usalo para recordar preferencias del usuario, datos personales, decisiones pasadas.\n"
}
