// Package agent — port.go define los contratos de módulos para el agente.
//
// Las capacidades se modelan con interfaces segregadas (Gen2):
//   - Module:    interfaz base (Name, Schema, Init)
//   - DataReader: módulos que proveen datos (read-only tools + PromptSection)
//   - DataWriter: módulos que mutan estado (write tools)
//
// Un mismo tipo puede implementar DataReader, DataWriter, ambos, o ninguno.
package agent

import (
	"context"
	"database/sql"

	"github.com/charmbracelet/log"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// ── Interfaces segregadas (Gen2) ─────────────────────────────────────────────

type Module interface {
	Name() string
	Schema() []string
	Init(deps PortDeps) error
}

type DataReader interface {
	Module
	PromptSection(ctx context.Context, userID string) string
	ReadTools() []agentllm.ToolDef
	Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error)
}

type DataWriter interface {
	Module
	WriteTools() []agentllm.ToolDef
	Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error)
}

type backgroundModule interface {
	Module
	Start(ctx context.Context) error
}

// ── PortDeps — recursos provisionados por el agente ──────────────────────────

// PortDeps son los recursos que el agente provisiona a cada módulo en Init.
type PortDeps struct {
	DB        PortDB
	Logger    *log.Logger
	LLM       agentllm.LLM
	Messenger Messenger

	// BackgroundLLM es un LLM alternativo para operaciones background (scheduler,
	// extractores, etc.). Si no se configura, se usa LLM por defecto.
	// Útil para reservar LLMs con capacidades especiales (Google Search Grounding)
	// solo para el chat principal, y usar LLMs más económicos para background.
	BackgroundLLM agentllm.LLM

	// Repos pre-construidos para uso directo en módulos.
	Config       domain.ConfigRepository
	Tasks        domain.TaskRepository
	Reminders    domain.ReminderRepository
	Scheduler    domain.SchedulerRepository
	Debts        domain.DebtRepository
	DebtPayments domain.DebtPaymentRepository
}

// PortDB — acceso controlado a la base de datos.
type PortDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
