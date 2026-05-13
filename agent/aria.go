// Package agent — aria.go es el orquestador central del sistema.
//
// Aria gestiona el flujo conversacional:
//   - Recibe mensajes de texto o audio
//   - Gestiona sesiones LLM por usuario (TTL 30min)
//   - Ejecuta tool calls en paralelo y entrega respuestas vía Messenger
//
// Aria NO contiene lógica de negocio. Los dominios viven en Module implementations.
package agent

import (
	"database/sql"
	"time"

	"github.com/charmbracelet/log"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// Aria orquesta el flujo conversacional.
type Aria struct {
	llm       agentllm.LLM
	messenger Messenger
	executor  *Executor
	db        *sql.DB
	repos     Repos
	sessions  *agentllm.SessionManager
	logger    *log.Logger
	registry  *ToolRegistry
	modules   []Module
}

// Repos agrupa los repositorios de dominio disponibles para los módulos.
type Repos struct {
	Tasks     domain.TaskRepository
	Reminders domain.ReminderRepository
}

// New crea una Aria con arquitectura de módulos.
func New(llm agentllm.LLM, db *sql.DB, logger *log.Logger, modules []Module, opts ...Option) *Aria {
	a := &Aria{
		llm:     llm,
		db:      db,
		logger:  logger.WithPrefix("Aria 🧠"),
		modules: modules,
	}
	for _, opt := range opts {
		opt(a)
	}

	a.registry = NewToolRegistry()
	a.provisionModules()

	a.executor = NewExecutor(a.registry, a.logger)
	a.sessions = agentllm.NewSessionManager(
		30*time.Minute,
		logger.GetLevel() == log.DebugLevel,
		logger,
	)
	return a
}

// Option configura Aria en construcción (functional options pattern).
type Option func(*Aria)

// WithMessenger inyecta un Messenger para la entrega de respuestas.
func WithMessenger(m Messenger) Option {
	return func(a *Aria) { a.messenger = m }
}

// WithRepos inyecta los repositorios de dominio para uso en módulos.
func WithRepos(r Repos) Option {
	return func(a *Aria) { a.repos = r }
}
