// Package agent — port.go define los contratos de módulos para el agente.
//
// Hay dos generaciones de interfaz:
//
//  1. DataPort (gen1): interfaz monolítica con Tools()/Handle()/PromptSection().
//  2. Module + DataReader + DataWriter (gen2): interfaces segregadas.
//
// El agente acepta ambas generaciones vía type assertions en provisionModules().
package agent

import (
	"context"
	"database/sql"

	"github.com/charmbracelet/log"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// ── Generación 1: DataPort ────────────────────────────────────────────────────

type DataPort interface {
	Name() string
	Schema() []string
	Init(deps PortDeps) error
	Tools() []agentllm.ToolDef
	Handle(ctx context.Context, tool string, args map[string]any) (map[string]any, error)
	PromptSection(ctx context.Context, userID string) string
}

// ── Generación 2: interfaces segregadas ──────────────────────────────────────

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

	// Repos pre-construidos para uso directo en módulos.
	Config    domain.ConfigRepository
	Tasks     domain.TaskRepository
	Reminders domain.ReminderRepository
}

// PortDB — acceso controlado a la base de datos.
type PortDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
