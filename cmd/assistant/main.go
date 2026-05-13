// Package main — entry point del asistente personal.
//
// Uso con Infisical:
//
//	infisical run -- go run ./cmd/assistant
//
// O directo con env vars:
//
//	export OPENAI_API_KEY=sk-...
//	export OPENAI_BASE_URL=https://openrouter.ai/api/v1
//	export OPENAI_MODEL=google/gemini-2.5-flash
//	export ALLOWED_NUMBERS=56912345678
//	go run ./cmd/assistant
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	charm "github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"

	"github.com/carlospereira5/PersonalAssistant/agent"
	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/db"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/tasks"
	"github.com/carlospereira5/PersonalAssistant/reminders"
	"github.com/carlospereira5/PersonalAssistant/whatsapp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logLevel := getEnv("LOG_LEVEL", "info")
	parsedLevel, err := charm.ParseLevel(logLevel)
	if err != nil {
		parsedLevel = charm.InfoLevel
	}
	logger := charm.NewWithOptions(os.Stderr, charm.Options{
		ReportCaller: parsedLevel >= charm.DebugLevel,
		Level:        parsedLevel,
	})

	// ── Config desde environment (Infisical inyecta estas vars) ──────────────
	openAIKey := getEnv("OPENAI_API_KEY", "")
	openAIBaseURL := getEnv("OPENAI_BASE_URL", "")
	openAIModel := getEnv("OPENAI_MODEL", "gpt-4o-mini")
	dbPath := getEnv("DATABASE_PATH", "./assistant.db")
	allowedRaw := getEnv("ALLOWED_NUMBERS", "")
	whatsAppDBPath := getEnv("WHATSAPP_DB_PATH", "./whatsapp.db")

	if openAIKey == "" {
		logger.Fatal("OPENAI_API_KEY es requerida — configurala en Infisical")
	}

	// ── LLM ───────────────────────────────────────────────────────────────────
	oc := openai.DefaultConfig(openAIKey)
	if openAIBaseURL != "" {
		oc.BaseURL = openAIBaseURL
	}
	llmClient := openai.NewClientWithConfig(oc)
	llm := agentllm.NewOpenAILLM(llmClient, openAIModel, openAIModel, nil)

	// ── Base de datos ─────────────────────────────────────────────────────────
	sqlite, err := db.NewDB(dbPath, logger.WithPrefix("DB"))
	if err != nil {
		logger.Fatal("Error abriendo base de datos", "err", err)
	}
	defer sqlite.Close()

	if err := sqlite.MigrateContext(ctx); err != nil {
		logger.Fatal("Error ejecutando migraciones", "err", err)
	}

	taskRepo := db.NewTaskRepo(sqlite)
	reminderRepo := db.NewReminderRepo(sqlite)
	configRepo := db.NewConfigRepo(sqlite)

	// ── Módulos ───────────────────────────────────────────────────────────────
	tasksModule := tasks.New()

	modules := []agent.Module{tasksModule}

	// ── Aria (orquestador) ────────────────────────────────────────────────────
	aria := agent.New(llm, sqlite.Db, logger, modules,
		agent.WithRepos(agent.Repos{
			Config:    configRepo,
			Tasks:     taskRepo,
			Reminders: reminderRepo,
		}),
	)

	// ── WhatsApp ──────────────────────────────────────────────────────────────
	allowedNumbers := strings.FieldsFunc(allowedRaw, func(r rune) bool { return r == ',' || r == ' ' })
	whatsAppBot, err := whatsapp.New(ctx, aria, whatsAppDBPath, allowedNumbers, "", logger, configRepo)
	if err != nil {
		logger.Fatal("Error creando bot de WhatsApp", "err", err)
	}

	whatsAppMessenger := whatsapp.NewMessenger(whatsAppBot)
	aria.SetMessenger(whatsAppMessenger)

	// ── Reminders service ─────────────────────────────────────────────────────
	reminderSvc := reminders.New(reminderRepo, configRepo, whatsAppMessenger, logger.WithPrefix("Reminders"))
	go reminderSvc.Start(ctx)

	// ── Start ─────────────────────────────────────────────────────────────────
	logger.Info("Asistente personal iniciado", "model", openAIModel)
	aria.Start(ctx)
	if err := whatsAppBot.Start(ctx); err != nil {
		logger.Error("WhatsApp bot stopped", "err", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
