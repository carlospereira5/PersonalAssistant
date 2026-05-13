// Package agent — aria_chat.go implementa el loop de conversación del agente.
package agent

import (
	"context"
	"fmt"
	"strings"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
)

// Chat envía un mensaje al LLM, ejecuta el loop de function calling
// y retorna la respuesta de texto final.
func (a *Aria) Chat(ctx context.Context, userID, message string) (string, error) {
	return a.ChatWithMessenger(ctx, userID, message, nil)
}

// ChatWithMessenger permite inyectar un Messenger específico para esta sesión.
func (a *Aria) ChatWithMessenger(ctx context.Context, userID, message string, m Messenger) (string, error) {
	a.logger.Info("Mensaje recibido", "user", userID, "msg_len", len(message))
	if m == nil {
		m = a.messenger
	}
	ctx = ctxWithUserID(ctx, userID)

	session, err := a.getSession(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("obteniendo sesión: %w", err)
	}

	text, calls, err := session.Send(ctx, message)
	if err != nil {
		a.logger.Error("Fallo en LLM Send", "err", err)
		return "", fmt.Errorf("LLM send: %w", err)
	}

	// Loop de function calling — máximo 5 iteraciones.
	for i := 0; i < 5 && len(calls) > 0; i++ {
		a.logger.Info("Ejecutando herramientas", "cantidad", len(calls))
		results, execErr := a.executor.Execute(ctx, calls, a.execute)
		if execErr != nil {
			return "", fmt.Errorf("ejecución de tools: %w", execErr)
		}
		text, calls, err = session.SendToolResults(ctx, results)
		if err != nil {
			return "", fmt.Errorf("tool results al LLM: %w", err)
		}
	}

	if text == "" {
		return "No pude generar una respuesta. Intentá reformular tu pregunta.", nil
	}
	a.logger.Info("Respuesta generada", "user", userID, "resp_len", len(text))
	return text, nil
}

// getSession recupera o crea la sesión LLM para un usuario.
// buildPrompt solo se invoca si la sesión no existe o expiró (TTL 30min).
// Para sesiones activas, el costo de buildPrompt (queries a DB incluidas) se evita.
func (a *Aria) getSession(ctx context.Context, userID string) (agentllm.Session, error) {
	if a.sessions.Exists(userID) {
		return a.sessions.GetOrCreate(ctx, userID, a.llm, "", a.registry.Tools())
	}
	prompt := a.buildPrompt(ctx, userID)
	return a.sessions.GetOrCreate(ctx, userID, a.llm, prompt, a.registry.Tools())
}

// buildPrompt construye el system prompt combinando:
//  1. Core del agente (persona, fecha, formato)
//  2. PromptSection de cada módulo DataReader (Gen2) o DataPort (Gen1)
func (a *Aria) buildPrompt(ctx context.Context, userID string) string {
	var b strings.Builder
	b.Grow(4000)

	b.WriteString(buildCorePrompt())

	for _, m := range a.modules {
		var section string
		if r, ok := m.(DataReader); ok {
			section = r.PromptSection(ctx, userID)
		} else if p, ok := m.(DataPort); ok {
			section = p.PromptSection(ctx, userID)
		}
		if section != "" {
			b.WriteString(section)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// execute es el dispatcher interno — rutea al módulo correcto via registry.
// Algunos modelos via OpenRouter prefijan los tool names con "default_api." — lo stripeamos.
func (a *Aria) execute(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	name = strings.TrimPrefix(name, "default_api.")
	a.logger.Debug("Ejecutando tool", "name", name, "args", fmt.Sprintf("%v", args))
	result, err := a.registry.Execute(ctx, name, args)
	if err != nil {
		return result, err
	}
	a.logger.Debug("Tool result", "name", name, "result", fmt.Sprintf("%v", result))
	return result, nil
}
