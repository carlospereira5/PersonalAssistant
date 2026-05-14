package search

import (
	"context"
	"net/http"
	"time"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
)

// Module implementa Gen2 DataReader para búsqueda web.
// web_search usa DuckDuckGo (gratuito, sin API key).
// fetch_url usa HTTP directo para leer páginas web.
type Module struct {
	client *http.Client
	logger *charm.Logger
}

// New crea un módulo de búsqueda web.
// web_search y fetch_url están siempre disponibles.
func New() *Module {
	return &Module{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (m *Module) Name() string     { return "search" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.logger = deps.Logger
	return nil
}

func (m *Module) PromptSection(ctx context.Context, userID string) string { return "" }
