package agent

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/charmbracelet/log"

	agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"
	"golang.org/x/sync/errgroup"
)

// Executor gestiona la ejecución paralela de herramientas.
type Executor struct {
	registry *ToolRegistry
	logger   *log.Logger
}

// NewExecutor crea un nuevo ejecutor.
func NewExecutor(reg *ToolRegistry, logger *log.Logger) *Executor {
	return &Executor{
		registry: reg,
		logger:   logger,
	}
}

// Execute ejecuta un conjunto de tool calls de forma concurrente.
// Utiliza errgroup para gestionar la cancelación y reportar el primer error fatal,
// aunque la mayoría de los errores de herramientas se capturan como resultados.
func (e *Executor) Execute(ctx context.Context, calls []agentllm.ToolCall, executeFn func(ctx context.Context, name string, args map[string]any) (map[string]any, error)) ([]agentllm.ToolResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	results := make([]agentllm.ToolResult, len(calls))
	g, ctx := errgroup.WithContext(ctx)

	for i, call := range calls {
		i, call := i, call // capturar para goroutine
		g.Go(func() error {
			// Recuperación de panics en herramientas para robustez
			defer func() {
				if r := recover(); r != nil {
					e.logger.Error("Panic en herramienta",
						"component", "agent",
						"tool", call.Name,
						"panic", fmt.Sprintf("%v", r),
					)
					results[i] = agentllm.ToolResult{
						Name:   call.Name,
						Result: map[string]any{"error": fmt.Sprintf("panic in tool: %v\n%s", r, debug.Stack())},
					}
				}
			}()

			startTime := time.Now()
			e.logger.Debug("Iniciando herramienta", "component", "agent", "tool", call.Name)
			result, err := executeFn(ctx, call.Name, call.Args)
			elapsed := time.Since(startTime)
			if err != nil {
				e.logger.Error("Fallo en herramienta",
					"component", "agent",
					"tool", call.Name,
					"elapsed", elapsed,
					"err", err,
				)
				results[i] = agentllm.ToolResult{
					Name:   call.Name,
					Result: map[string]any{"error": err.Error()},
				}
			} else {
				e.logger.Debug("Herramienta completada", "component", "agent", "tool", call.Name, "elapsed", elapsed)
				results[i] = agentllm.ToolResult{
					Name:   call.Name,
					Result: result,
				}
			}
			return nil // No devolvemos error fatal para no detener otras herramientas
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	return results, nil
}
