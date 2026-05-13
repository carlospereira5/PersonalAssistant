package scheduler

import (
	"context"
	"fmt"
	"time"
)

func (m *Module) Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "create_routine":
		return m.createRoutine(ctx, args)
	case "delete_routine":
		return m.deleteRoutine(ctx, args)
	case "pause_routine":
		return m.pauseRoutine(ctx, args)
	case "resume_routine":
		return m.resumeRoutine(ctx, args)
	}
	return nil, fmt.Errorf("scheduler: herramienta desconocida %q", tool)
}

func (m *Module) createRoutine(ctx context.Context, args map[string]any) (map[string]any, error) {
	cronExpr, _ := args["cron"].(string)
	prompt, _ := args["prompt"].(string)

	if cronExpr == "" {
		return nil, fmt.Errorf("create_routine: cron expression requerida")
	}
	if prompt == "" {
		return nil, fmt.Errorf("create_routine: prompt requerido")
	}

	schedule, err := m.cronParser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("create_routine: expresión cron inválida %q: %w", cronExpr, err)
	}

	id, err := m.repo.Create(ctx, cronExpr, prompt)
	if err != nil {
		return nil, fmt.Errorf("create_routine: %w", err)
	}

	nextRun := schedule.Next(time.Now())
	m.logger.Info("Rutina creada", "id", id, "cron", cronExpr)

	return map[string]any{
		"id":       id,
		"status":   "active",
		"cron":     cronExpr,
		"next_run": nextRun.Format(time.RFC3339),
	}, nil
}

func (m *Module) deleteRoutine(ctx context.Context, args map[string]any) (map[string]any, error) {
	id := int64(args["id"].(float64))
	if err := m.repo.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("delete_routine: %w", err)
	}
	m.logger.Info("Rutina eliminada", "id", id)
	return map[string]any{"ok": true, "id": id}, nil
}

func (m *Module) pauseRoutine(ctx context.Context, args map[string]any) (map[string]any, error) {
	id := int64(args["id"].(float64))
	if err := m.repo.UpdateStatus(ctx, id, "paused"); err != nil {
		return nil, fmt.Errorf("pause_routine: %w", err)
	}
	m.logger.Info("Rutina pausada", "id", id)
	return map[string]any{"ok": true, "id": id, "status": "paused"}, nil
}

func (m *Module) resumeRoutine(ctx context.Context, args map[string]any) (map[string]any, error) {
	id := int64(args["id"].(float64))
	if err := m.repo.UpdateStatus(ctx, id, "active"); err != nil {
		return nil, fmt.Errorf("resume_routine: %w", err)
	}
	m.logger.Info("Rutina reanudada", "id", id)
	return map[string]any{"ok": true, "id": id, "status": "active"}, nil
}
