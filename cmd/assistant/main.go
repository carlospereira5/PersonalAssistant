// Package main — entry point del asistente personal.
//
// Uso:
//
//	export OPENAI_API_KEY=sk-...
//	export OPENAI_MODEL=gpt-4o-mini
//	export DATABASE_PATH=./assistant.db
//	export ADMIN_JID=56912345678@s.whatsapp.net
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

	logger := charm.NewWithOptions(os.Stderr, charm.Options{
		ReportCaller: true,
		Level:        charm.InfoLevel,
	})

	// ── Config desde environment ──────────────────────────────────────────────
	openAIKey := getEnv("OPENAI_API_KEY", "")
	openAIModel := getEnv("OPENAI_MODEL", "gpt-4o-mini")
	dbPath := getEnv("DATABASE_PATH", "./assistant.db")
	adminJID := getEnv("ADMIN_JID", "")
	allowedRaw := getEnv("ALLOWED_NUMBERS", "")
	whatsAppDBPath := getEnv("WHATSAPP_DB_PATH", "./whatsapp.db")

	if openAIKey == "" {
		logger.Fatal("OPENAI_API_KEY es requerida")
	}
	if adminJID == "" {
		logger.Fatal("ADMIN_JID es requerida (ej: 56912345678@s.whatsapp.net)")
	}

	// ── LLM ───────────────────────────────────────────────────────────────────
	llmClient := openai.NewClient(openAIKey)
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

	// ── Módulos ───────────────────────────────────────────────────────────────
	tasksModule := tasks.New()

	modules := []agent.Module{tasksModule}

	// ── Aria (orquestador) ────────────────────────────────────────────────────
	aria := agent.New(llm, sqlite.Db, logger, modules,
		agent.WithRepos(agent.Repos{
			Tasks:     taskRepo,
			Reminders: reminderRepo,
		}),
	)

	// ── WhatsApp ──────────────────────────────────────────────────────────────
	allowedNumbers := strings.FieldsFunc(allowedRaw, func(r rune) bool { return r == ',' || r == ' ' })
	whatsAppBot, err := whatsapp.New(ctx, aria, whatsAppDBPath, allowedNumbers, "", logger)
	if err != nil {
		logger.Fatal("Error creando bot de WhatsApp", "err", err)
	}

	whatsAppMessenger := whatsapp.NewMessenger(whatsAppBot)
	aria.SetMessenger(whatsAppMessenger)

	// ── Reminders service ─────────────────────────────────────────────────────
	reminderSvc := reminders.New(reminderRepo, whatsAppMessenger, logger.WithPrefix("Reminders"), adminJID)
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
