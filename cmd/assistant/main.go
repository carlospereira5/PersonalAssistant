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
//	export GROQ_API_KEY=gsk_...           (opcional — para transcripción de audio)
//	export ALLOWED_NUMBERS=56912345678
//	go run ./cmd/assistant
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	charm "github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"

	"github.com/carlospereira5/PersonalAssistant/agent"
	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"github.com/carlospereira5/PersonalAssistant/internal/db"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/debts"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/memory"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/scheduler"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/search"
	"github.com/carlospereira5/PersonalAssistant/internal/modules/tasks"
	"github.com/carlospereira5/PersonalAssistant/reminders"
	"github.com/carlospereira5/PersonalAssistant/whatsapp"
)

// DNS servers para el resolver override.
// Google DNS primario, Cloudflare como fallback.
var dnsServers = []string{"8.8.8.8:53", "1.1.1.1:53"}

// init reemplaza el resolver DNS por defecto para evitar depender del proxy DNS
// de Tailscale (127.0.0.1:53). En Android+Tailscale, ese proxy se cae cuando el
// dispositivo entra en deep sleep, dejando a whatsmeow sin poder reconectar.
//
// Con PreferGo:true + Dial directo, toda resolución DNS en la aplicación
// (whatsmeow, OpenRouter, Groq, etc.) va a los servidores definidos arriba sin
// pasar por el resolver del sistema (/etc/resolv.conf o getaddrinfo).
//
// RIESGO: esto IGNORA servidores DNS locales. No puede resolver dominios de la
// tailnet (MagicDNS) ni dominios .local. Es intencional — PersonalAssistant solo
// necesita resolver dominios públicos.
func init() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			// Intenta cada DNS server en orden hasta que uno responda.
			for _, srv := range dnsServers {
				conn, err := d.DialContext(ctx, "udp", srv)
				if err == nil {
					return conn, nil
				}
			}
			return nil, fmt.Errorf("dns: all servers unreachable (%v)", dnsServers)
		},
	}
}

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
	groqAPIKey := getEnv("GROQ_API_KEY", "")
	dbPath := getEnv("DATABASE_PATH", "./assistant.db")
	allowedRaw := getEnv("ALLOWED_NUMBERS", "")
	whatsAppDBPath := getEnv("WHATSAPP_DB_PATH", "./whatsapp.db")

	if openAIKey == "" {
		logger.Fatal("OPENAI_API_KEY es requerida — configurala en Infisical")
	}

	// ── LLM: TODO via OpenRouter ─────────────────────────────────────────────
	// Usamos OpenRouter para todo. El modelo google/gemini-2.5-flash se sirve
	// via OpenRouter, sin llamar a la API nativa de Google.
	// Esto evita el cuoteo del free tier de Gemini (20 requests/día).
	oc := openai.DefaultConfig(openAIKey)
	if openAIBaseURL != "" {
		oc.BaseURL = openAIBaseURL
	}
	llmClient := openai.NewClientWithConfig(oc)

	// Whisper (transcripción de audio) via Groq — gratis, sin cuota relevante.
	// OpenRouter no soporta el endpoint de audio, por eso usamos Groq.
	var whisperClient *openai.Client
	if groqAPIKey != "" {
		groqCfg := openai.DefaultConfig(groqAPIKey)
		groqCfg.BaseURL = "https://api.groq.com/openai/v1"
		whisperClient = openai.NewClientWithConfig(groqCfg)
		logger.Info("Whisper: Groq (transcripción de audio)")
	}

	llm := agentllm.NewOpenAILLM(llmClient, openAIModel, openAIModel, whisperClient)
	logger.Info("LLM: OpenRouter", "model", openAIModel)

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
	schedulerRepo := db.NewSchedulerRepo(sqlite)

	debtRepo := db.NewDebtRepo(sqlite)
	paymentRepo := db.NewDebtPaymentRepo(sqlite)

	// ── Módulos ───────────────────────────────────────────────────────────────
	tasksModule := tasks.New()

	searchModule := search.New()

	schedulerModule := scheduler.New()

	memoryModule := memory.New()

	debtsModule := debts.New()

	modules := []agent.Module{tasksModule, searchModule, schedulerModule, memoryModule, debtsModule}

	// ── Aria (orquestador) ────────────────────────────────────────────────────
	aria := agent.New(llm, sqlite.Db, logger, modules,
		agent.WithRepos(agent.Repos{
			Config:       configRepo,
			Tasks:        taskRepo,
			Reminders:    reminderRepo,
			Scheduler:    schedulerRepo,
			Debts:        debtRepo,
			DebtPayments: paymentRepo,
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
