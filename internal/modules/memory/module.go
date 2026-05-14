// Package memory — módulo Gen2 DataReader + DataWriter para memoria semántica.
//
// Expone tools para que el LLM guarde y recupere hechos (facts) de forma
// persistente entre sesiones, usando SQLite FTS5 para búsqueda full-text.
package memory

import (
	"context"
	"fmt"
	"strings"

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

// PromptSection inyecta los facts persistidos directamente en el system prompt.
// El LLM recibe estos hechos como contexto desde el primer mensaje, sin
// necesidad de llamar herramientas de memoria para descubrirlos.
//
// Las herramientas (save_fact, get_fact, search_memory) ya están definidas
// via tool definitions — esta sección solo agrega los valores concretos.
func (m *Module) PromptSection(ctx context.Context, _ string) string {
	facts, err := m.store.GetAllFacts(ctx)
	if err != nil {
		m.logger.Warn("No se pudieron cargar facts para el prompt", "err", err)
	}

	var b strings.Builder
	b.WriteString("\n## MEMORIA PERSISTENTE\n")

	if len(facts) == 0 {
		b.WriteString("\nAún no hay información guardada. Usá save_fact para recordar datos entre sesiones.\n")
		return b.String()
	}

	b.WriteString("\nInformación recordada de sesiones anteriores (DEBES usarla en tus respuestas):\n")
	for _, f := range facts {
		b.WriteString(fmt.Sprintf("- %s: %s\n", f.Key, f.Value))
	}
	b.WriteString("\nUsá esta información automáticamente. Si el usuario te da información nueva, guardala con save_fact.\n")

	return b.String()
}
