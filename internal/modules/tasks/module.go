// Package tasks — módulo de gestión de tareas para el agente.
package tasks

import (
	"context"
	"fmt"
	"strings"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// Module gestiona el CRUD de tareas vía herramientas del agente.
// Implementa las interfaces agent.DataReader y agent.DataWriter (Gen2).
type Module struct {
	repo      domain.TaskRepository
	reminders domain.ReminderRepository
	db        agent.PortDB
	logger    *charm.Logger
}

// New crea un nuevo módulo de tareas.
func New() *Module { return &Module{} }

func (m *Module) Name() string     { return "tasks" }
func (m *Module) Schema() []string { return nil }

func (m *Module) Init(deps agent.PortDeps) error {
	m.repo = deps.Tasks
	m.reminders = deps.Reminders
	m.db = deps.DB
	m.logger = deps.Logger
	return nil
}

// PromptSection inyecta las tareas activas en el system prompt del agente.
func (m *Module) PromptSection(ctx context.Context, _ string) string {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, task_name, deadline, status
		FROM tasks
		WHERE status IN ('PENDING', 'IN_PROGRESS')
		ORDER BY deadline ASC
	`)
	if err != nil {
		return ""
	}
	defer rows.Close()

	type activeTask struct {
		id       int64
		name     string
		deadline string
		status   string
	}

	var pending []activeTask
	for rows.Next() {
		var at activeTask
		if err := rows.Scan(&at.id, &at.name, &at.deadline, &at.status); err != nil {
			continue
		}
		pending = append(pending, at)
	}
	if len(pending) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## TAREAS PENDIENTES\n")
	for _, at := range pending {
		fmt.Fprintf(&b, "- ID %d: %s (vence %s, estado: %s)\n",
			at.id, at.name, at.deadline[:10], at.status)
	}
	return b.String()
}
