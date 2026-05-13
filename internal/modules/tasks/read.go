package tasks

import (
	"context"
	"fmt"
	"time"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "get_all_tasks":
		tasks, err := m.repo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("get_all_tasks: %w", err)
		}
		return map[string]any{"tasks": marshalTasks(tasks), "total": len(tasks)}, nil

	case "get_current_time":
		return m.getCurrentTime(ctx)
	}
	return nil, fmt.Errorf("tasks: herramienta desconocida %q", tool)
}

// getCurrentTime retorna la hora actual en UTC y en America/Santiago.
func (m *Module) getCurrentTime(ctx context.Context) (map[string]any, error) {
	santiagoLoc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		santiagoLoc = time.FixedZone("CLT", -4*60*60)
	}

	now := time.Now()
	local := now.In(santiagoLoc)

	return map[string]any{
		"utc":         now.Format(time.RFC3339),
		"local":       local.Format(time.RFC3339),
		"timezone":    "America/Santiago",
		"utc_offset":  local.Format("-07:00"),
		"weekday":     local.Weekday().String(),
		"iso_weekday": int(local.Weekday()),
	}, nil
}
