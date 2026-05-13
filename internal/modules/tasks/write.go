package tasks

import (
	"context"
	"fmt"
)

func (m *Module) Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "create_task":
		name, _ := args["name"].(string)
		deadline, _ := args["deadline"].(string)
		description, _ := args["description"].(string)

		id, err := m.repo.Create(ctx, name, deadline, description)
		if err != nil {
			return nil, fmt.Errorf("create_task: %w", err)
		}
		m.logger.Info("Tarea creada", "id", id, "name", name)

		// Siempre crear un recordatorio automático para el deadline.
		if _, err := m.reminders.Create(ctx, id, deadline); err != nil {
			m.logger.Warn("Error guardando recordatorio automático del deadline", "task_id", id, "remind_at", deadline, "err", err)
		}

		// Guardar recordatorios adicionales si se especificaron.
		if raw, ok := args["reminders"]; ok {
			if list, ok := raw.([]any); ok {
				for _, item := range list {
					ts, ok := item.(string)
					if !ok || ts == "" {
						continue
					}
					if _, err := m.reminders.Create(ctx, id, ts); err != nil {
						m.logger.Warn("Error guardando recordatorio extra", "task_id", id, "remind_at", ts, "err", err)
					}
				}
			}
		}

		return map[string]any{"ok": true, "id": id, "name": name, "deadline": deadline}, nil

	case "update_task_status":
		id := int64(args["id"].(float64))
		status, _ := args["status"].(string)
		if err := m.repo.UpdateStatus(ctx, id, status); err != nil {
			return nil, fmt.Errorf("update_task_status: %w", err)
		}
		return map[string]any{"ok": true, "id": id, "status": status}, nil

	case "delete_task":
		id := int64(args["id"].(float64))
		if err := m.repo.Delete(ctx, id); err != nil {
			return nil, fmt.Errorf("delete_task: %w", err)
		}
		return map[string]any{"ok": true, "id": id}, nil
	}
	return nil, fmt.Errorf("tasks: herramienta desconocida %q", tool)
}
