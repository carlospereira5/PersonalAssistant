package agent

import (
	"context"
	"fmt"
	"sync"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
)

// ToolHandler procesa una tool call del LLM.
type ToolHandler func(ctx context.Context, args map[string]any) (map[string]any, error)

// ToolRegistry registra tool definitions + handlers. Permite que módulos externos
// registren sus propias herramientas sin modificar el agent core.
//
// Thread-safe: Register puede llamarse desde múltiples goroutines.
type ToolRegistry struct {
	mu       sync.RWMutex
	defs     []agentllm.ToolDef
	handlers map[string]ToolHandler
}

// NewToolRegistry crea un ToolRegistry vacío.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		handlers: make(map[string]ToolHandler),
	}
}

// Register agrega una herramienta al registro.
// Si ya existe una tool con el mismo nombre, la reemplaza.
func (r *ToolRegistry) Register(def agentllm.ToolDef, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, d := range r.defs {
		if d.Name == def.Name {
			r.defs[i] = def
			r.handlers[def.Name] = handler
			return
		}
	}
	r.defs = append(r.defs, def)
	r.handlers[def.Name] = handler
}

// Tools retorna una copia de los ToolDef registrados.
// Safe to pass directly to a LLM session.
func (r *ToolRegistry) Tools() []agentllm.ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]agentllm.ToolDef, len(r.defs))
	copy(out, r.defs)
	return out
}

// Execute despacha una tool call al handler registrado.
// Retorna error si no existe un handler para name.
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	r.mu.RLock()
	h, ok := r.handlers[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("herramienta desconocida: %q", name)
	}
	return h(ctx, args)
}
