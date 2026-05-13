package tasks

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "get_all_tasks":
		tasks, err := m.repo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("get_all_tasks: %w", err)
		}
		return map[string]any{"tasks": marshalTasks(tasks), "total": len(tasks)}, nil
	}
	return nil, fmt.Errorf("tasks: herramienta desconocida %q", tool)
}
