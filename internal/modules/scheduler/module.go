package scheduler

import (
	"context"
	"fmt"
	"strings"

	charm "github.com/charmbracelet/log"
	"github.com/robfig/cron/v3"

	"github.com/carlospereira5/PersonalAssistant/agent"
	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

type Module struct {
	repo       domain.SchedulerRepository
	configRepo domain.ConfigRepository
	llm        agentllm.LLM
	bgLLM      agentllm.LLM // LLM para ejecución background (usa BackgroundLLM si está configurada)
	messenger  agent.Messenger
	cronParser cron.Parser
	logger     *charm.Logger
}

func New() *Module { return &Module{} }

func (m *Module) Name() string     { return "scheduler" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.repo = deps.Scheduler
	m.configRepo = deps.Config
	m.llm = deps.LLM
	m.bgLLM = deps.LLM // default: mismo LLM que el chat principal
	if deps.BackgroundLLM != nil {
		m.bgLLM = deps.BackgroundLLM // si hay uno específico para background, usarlo
	}
	m.messenger = deps.Messenger
	m.logger = deps.Logger
	m.cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return nil
}

func (m *Module) PromptSection(ctx context.Context, _ string) string {
	routines, err := m.repo.GetAll(ctx)
	if err != nil {
		return ""
	}

	var activeCount, pausedCount int
	for _, r := range routines {
		switch r.Status {
		case "active":
			activeCount++
		case "paused":
			pausedCount++
		}
	}
	if activeCount == 0 && pausedCount == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n## RUTINAS PROGRAMADAS\n")
	if activeCount > 0 {
		fmt.Fprintf(&b, "Activas: %d\n", activeCount)
	}
	if pausedCount > 0 {
		fmt.Fprintf(&b, "Pausadas: %d\n", pausedCount)
	}
	b.WriteString("Usá create_routine(cron, prompt) para crear nuevas.\n")
	b.WriteString("Formato cron: \"minuto hora día-del-mes mes día-de-la-semana\".\n")
	b.WriteString("Ej: \"0 8 * * *\" = todos los días a las 08:00.\n")
	return b.String()
}
