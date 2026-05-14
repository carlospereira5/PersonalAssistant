// Package agent — hooks.go define el sistema de lifecycle hooks del agente.
//
// Los hooks permiten que código externo (memoria, observabilidad, logging)
// reaccione a eventos del ciclo de vida del agente sin modificar agent/.
// Son el equivalente a middleware en HTTP, pero para el loop agente.
package agent

import (
	"context"
	"sync"
)

// AgentHook es el punto de extensión del agente.
// Cualquier componente puede implementar esta interfaz para
// reaccionar a eventos del ciclo de vida.
type AgentHook interface {
	// OnMessage se llama antes de enviar el mensaje al LLM.
	// Útil para: logging, modificar el mensaje, inyectar contexto adicional.
	// Retorna el mensaje (posiblemente modificado) y un error.
	OnMessage(ctx context.Context, userID string, message string) (string, error)

	// OnResponse se llama después de cada respuesta del LLM.
	// Útil para: extraer facts, logging, detectar intenciones, conteo de tokens.
	OnResponse(ctx context.Context, userID string, response string) error

	// OnToolResult se llama después de cada tool call exitosa.
	// Útil para: actualizar memoria con resultados, auditoría.
	OnToolResult(ctx context.Context, userID string, tool string, args map[string]any, result map[string]any) error

	// OnSessionEnd se llama al finalizar cada turno de conversación.
	// Útil para: extracción batch de facts, cleanup.
	OnSessionEnd(ctx context.Context, userID string) error
}

// HookRegistry gestiona el registro y dispatch de hooks de forma thread-safe.
// Si no hay hooks registrados, las llamadas son no-op (sin overhead).
type HookRegistry struct {
	mu    sync.RWMutex
	hooks []AgentHook
}

// NewHookRegistry crea un HookRegistry vacío.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// Register agrega un hook al registry.
// Los hooks se ejecutan en orden de registro.
func (r *HookRegistry) Register(hook AgentHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// OnMessage dispatches OnMessage a todos los hooks registrados.
// Si un hook retorna error, el mensaje original se mantiene y se loguea el error.
// Si algún hook modifica el mensaje, se pasa la versión modificada al siguiente hook.
func (r *HookRegistry) OnMessage(ctx context.Context, userID string, message string) (string, error) {
	r.mu.RLock()
	hooks := r.hooks
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return message, nil
	}

	current := message
	for _, h := range hooks {
		modified, err := h.OnMessage(ctx, userID, current)
		if err != nil {
			return current, err
		}
		if modified != "" {
			current = modified
		}
	}
	return current, nil
}

// OnResponse dispatches OnResponse a todos los hooks registrados.
// Los errores se acumulan pero no interrumpen la cadena.
func (r *HookRegistry) OnResponse(ctx context.Context, userID string, response string) error {
	r.mu.RLock()
	hooks := r.hooks
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.OnResponse(ctx, userID, response); err != nil {
			return err
		}
	}
	return nil
}

// OnToolResult dispatches OnToolResult a todos los hooks registrados.
func (r *HookRegistry) OnToolResult(ctx context.Context, userID string, tool string, args map[string]any, result map[string]any) error {
	r.mu.RLock()
	hooks := r.hooks
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.OnToolResult(ctx, userID, tool, args, result); err != nil {
			return err
		}
	}
	return nil
}

// OnSessionEnd dispatches OnSessionEnd a todos los hooks registrados.
func (r *HookRegistry) OnSessionEnd(ctx context.Context, userID string) error {
	r.mu.RLock()
	hooks := r.hooks
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.OnSessionEnd(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}
