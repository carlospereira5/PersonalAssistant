package scheduler

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "list_routines":
		routines, err := m.repo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("list_routines: %w", err)
		}
		return map[string]any{"routines": marshalRoutines(routines), "total": len(routines)}, nil
	}
	return nil, fmt.Errorf("scheduler: herramienta desconocida %q", tool)
}
