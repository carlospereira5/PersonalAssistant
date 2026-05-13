package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"
)

// retryAfterRe parsea el tiempo sugerido en mensajes 429 del tipo:
// "Please try again in 49.63s."
var retryAfterRe = regexp.MustCompile(`try again in (\d+\.?\d*)s`)

// ── SessionManager ────────────────────────────────────────────────────────────

// managedSession encapsula una sesión LLM con su timestamp de último uso.
type managedSession struct {
	Session  Session
	LastUsed time.Time
}

// SessionManager gestiona el ciclo de vida de sesiones multi-turno en memoria.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*managedSession
	ttl      time.Duration
	debug    bool
	logger   *log.Logger
}

// NewSessionManager crea un manager con el TTL especificado.
func NewSessionManager(ttl time.Duration, debug bool, logger *log.Logger) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*managedSession),
		ttl:      ttl,
		debug:    debug,
		logger:   logger,
	}
}

// GetOrCreate recupera una sesión existente o crea una nueva si no existe o expiró.
func (sm *SessionManager) GetOrCreate(ctx context.Context, userID string, llm LLM, systemPrompt string, tools []ToolDef) (Session, error) {
	sm.mu.RLock()
	ms, exists := sm.sessions[userID]
	sm.mu.RUnlock()

	now := time.Now()

	if exists && now.Sub(ms.LastUsed) < sm.ttl {
		sm.mu.Lock()
		ms.LastUsed = now
		sm.mu.Unlock()
		return ms.Session, nil
	}

	newSession, err := llm.NewSession(ctx, systemPrompt, tools)
	if err != nil {
		return nil, err
	}

	// Envolvemos la sesión con tolerancia a fallos.
	newSession = WrapSessionWithLogger(newSession, sm.debug, sm.logger)

	sm.mu.Lock()
	sm.sessions[userID] = &managedSession{Session: newSession, LastUsed: now}
	sm.mu.Unlock()

	return newSession, nil
}

// Exists informa si hay una sesión activa (dentro de TTL) para el usuario dado.
// Se usa para evitar reconstruir el system prompt cuando la sesión ya existe.
func (sm *SessionManager) Exists(userID string) bool {
	sm.mu.RLock()
	ms, ok := sm.sessions[userID]
	sm.mu.RUnlock()
	return ok && time.Since(ms.LastUsed) < sm.ttl
}

// Delete elimina la sesión de un usuario, forzando una sesión nueva en el próximo Chat.
func (sm *SessionManager) Delete(userID string) {
	sm.mu.Lock()
	delete(sm.sessions, userID)
	sm.mu.Unlock()
}

// ── retrySession ─────────────────────────────────────────────────────────────

// retrySession envuelve una Session con lógica de reintento exponencial.
type retrySession struct {
	inner      Session
	maxRetries int
	debug      bool
	logger     *log.Logger
}

// WrapSession crea un decorador para reintentar errores de red o rate limits.
// Mantiene compatibilidad con tests que no inyectan logger.
func WrapSession(inner Session, debug bool) Session {
	return &retrySession{inner: inner, maxRetries: 3, debug: debug}
}

// WrapSessionWithLogger crea un decorador con logger charm inyectado.
func WrapSessionWithLogger(inner Session, debug bool, logger *log.Logger) Session {
	return &retrySession{inner: inner, maxRetries: 3, debug: debug, logger: logger}
}

func (s *retrySession) InjectAssistantMessage(content string) {
	s.inner.InjectAssistantMessage(content)
}

func (s *retrySession) Send(ctx context.Context, message string) (string, []ToolCall, error) {
	return s.withRetry(ctx, "Send", func() (string, []ToolCall, error) {
		return s.inner.Send(ctx, message)
	})
}

func (s *retrySession) Stream(ctx context.Context, message string) (<-chan StreamEvent, error) {
	// Para streaming no aplicamos reintentos automáticos de forma simple, delegamos al inner.
	return s.inner.Stream(ctx, message)
}

func (s *retrySession) SendToolResults(ctx context.Context, results []ToolResult) (string, []ToolCall, error) {
	return s.withRetry(ctx, "SendToolResults", func() (string, []ToolCall, error) {
		return s.inner.SendToolResults(ctx, results)
	})
}

func (s *retrySession) StreamToolResults(ctx context.Context, results []ToolResult) (<-chan StreamEvent, error) {
	return s.inner.StreamToolResults(ctx, results)
}

func (s *retrySession) withRetry(ctx context.Context, opName string, op func() (string, []ToolCall, error)) (string, []ToolCall, error) {
	var (
		text  string
		calls []ToolCall
		err   error
	)
	backoff := 4 * time.Second

	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		text, calls, err = op()
		if err == nil {
			return text, calls, nil
		}
		if !s.isRetryable(err) {
			return text, calls, err
		}
		if attempt == s.maxRetries {
			break
		}

		// Para 429: respetar el tiempo sugerido por la API (ej. "try again in 49.63s").
		// Exponential backoff solo aplica si el tiempo sugerido es menor.
		wait := backoff
		if suggested := parseRetryAfter(err); suggested > wait {
			wait = suggested
		}

		if s.debug && s.logger != nil {
			s.logger.Warn("Reintentando operación LLM",
				"component", "llm",
				"op", opName,
				"attempt", attempt,
				"max", s.maxRetries,
				"wait", wait,
				"err", err,
			)
		}
		select {
		case <-ctx.Done():
			return text, calls, ctx.Err()
		case <-time.After(wait):
			backoff *= 2
		}
	}
	return text, calls, fmt.Errorf("fallo final tras %d intentos: %w", s.maxRetries, err)
}

// parseRetryAfter extrae el tiempo sugerido de espera de un error 429.
// Soporta el formato de Groq/OpenAI: "Please try again in 49.63s."
// Agrega 1s de margen para evitar falsos negativos.
func parseRetryAfter(err error) time.Duration {
	var apiErr *openai.APIError
	if !errors.As(err, &apiErr) {
		return 0
	}
	m := retryAfterRe.FindStringSubmatch(apiErr.Message)
	if len(m) < 2 {
		return 0
	}
	secs, parseErr := strconv.ParseFloat(m[1], 64)
	if parseErr != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs*float64(time.Second)) + time.Second
}

func (s *retrySession) isRetryable(err error) bool {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.HTTPStatusCode
		return code == 429 || code >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// Gemini / Google API errors no tienen un tipo exportado accesible directamente.
	// Detectamos por el mensaje de error, que incluye el código HTTP o el error gRPC.
	msg := strings.ToUpper(err.Error())
	patterns := []string{
		"429",
		"RESOURCE_EXHAUSTED",
		"QUOTA",
		"503",
		"UNAVAILABLE",
		"INTERNAL",
		"500",
	}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
