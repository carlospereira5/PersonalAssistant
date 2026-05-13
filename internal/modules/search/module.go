package search

import (
	"context"
	"net/http"
	"time"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
)

type Module struct {
	client *http.Client
	logger *charm.Logger
}

func New() *Module { return &Module{} }

func (m *Module) Name() string     { return "search" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.logger = deps.Logger
	m.client = &http.Client{
		Timeout: 15 * time.Second,
	}
	return nil
}

func (m *Module) PromptSection(ctx context.Context, userID string) string { return "" }
