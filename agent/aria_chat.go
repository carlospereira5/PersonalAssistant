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

	// Hook: pre-procesamiento del mensaje antes de enviarlo al LLM.
	if a.hooks != nil {
		var err error
		message, err = a.hooks.OnMessage(ctx, userID, message)
		if err != nil {
			a.logger.Warn("OnMessage hook falló", "err", err)
			// Continuamos con el mensaje original — los hooks no bloquean el flujo.
		}
	}

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

	// Hook: post-procesamiento de la respuesta del LLM.
	if a.hooks != nil {
		if err := a.hooks.OnResponse(ctx, userID, text); err != nil {
			a.logger.Warn("OnResponse hook falló", "err", err)
		}
	}

	// Loop de function calling — máximo 5 iteraciones.
	for i := 0; i < 5 && len(calls) > 0; i++ {
		a.logger.Info("Ejecutando herramientas", "cantidad", len(calls))
		results, execErr := a.executor.Execute(ctx, calls, a.execute)
		if execErr != nil {
			return "", fmt.Errorf("ejecución de tools: %w", execErr)
		}

		// Hook: post-procesamiento de cada tool result.
		// Pareamos results con calls por índice para acceder a los args originales.
		if a.hooks != nil {
			for i, res := range results {
				args := calls[i].Args
				if err := a.hooks.OnToolResult(ctx, userID, res.Name, args, res.Result); err != nil {
					a.logger.Warn("OnToolResult hook falló", "tool", res.Name, "err", err)
				}
			}
		}

		text, calls, err = session.SendToolResults(ctx, results)
		if err != nil {
			return "", fmt.Errorf("tool results al LLM: %w", err)
		}

		// Hook: post-procesamiento de cada respuesta intermedia del LLM.
		if a.hooks != nil {
			if err := a.hooks.OnResponse(ctx, userID, text); err != nil {
				a.logger.Warn("OnResponse hook falló", "err", err)
			}
		}
	}

	if text == "" {
		text = "No pude generar una respuesta. Intentá reformular tu pregunta."
	}
	a.logger.Info("Respuesta generada", "user", userID, "resp_len", len(text))

	// Hook: fin del turno de conversación.
	if a.hooks != nil {
		if err := a.hooks.OnSessionEnd(ctx, userID); err != nil {
			a.logger.Warn("OnSessionEnd hook falló", "err", err)
		}
	}

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
//  2. PromptSection de cada módulo DataReader
func (a *Aria) buildPrompt(ctx context.Context, userID string) string {
	var b strings.Builder
	b.Grow(4000)

	b.WriteString(buildCorePrompt())

	for _, m := range a.modules {
		if r, ok := m.(DataReader); ok {
			if section := r.PromptSection(ctx, userID); section != "" {
				b.WriteString(section)
				b.WriteString("\n")
			}
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
