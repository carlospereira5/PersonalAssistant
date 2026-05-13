package search

import (
	"context"
	"net/http"
	"time"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
)

// Module implementa Gen2 DataReader/DataWriter para búsqueda web.
// web_search usa Gemini Search Grounding vía REST API.
// fetch_url usa HTTP directo.
type Module struct {
	client   *http.Client
	searcher *SearchClient
	logger   *charm.Logger
}

// New crea un módulo de búsqueda. Si apiKey y model son vacíos,
// web_search no estará disponible (fetch_url sí).
func New(apiKey, model string) *Module {
	m := &Module{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if apiKey != "" && model != "" {
		m.searcher = NewSearchClient(apiKey, model)
	}
	return m
}

func (m *Module) Name() string     { return "search" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.logger = deps.Logger
	return nil
}

func (m *Module) PromptSection(ctx context.Context, userID string) string { return "" }
