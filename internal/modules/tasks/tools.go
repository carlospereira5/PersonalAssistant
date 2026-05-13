package tasks

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "get_all_tasks",
			Description: "Retorna todas las tareas con su estado (PENDING, IN_PROGRESS, DONE, CANCELLED).",
		},
		{
			Name:        "get_current_time",
			Description: "Obtiene la fecha y hora actual en UTC y en la zona horaria del usuario (America/Santiago). Útil para resolver fechas, calcular deadlines y convertir entre zonas horarias sin depender del conocimiento interno del modelo.",
		},
	}
}

func (m *Module) WriteTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "create_task",
			Description: "Crea una nueva tarea. deadline en formato ISO8601. Si se especifican reminders, se enviará una notificación al usuario en esas fechas.",
			Parameters: []agentllm.ParamDef{
				{Name: "name", Type: "string", Description: "Nombre o descripción corta de la tarea."},
				{Name: "deadline", Type: "string", Description: "Fecha límite en formato ISO8601 (ej: 2026-05-14T00:00:00Z)."},
				{Name: "description", Type: "string", Description: "Descripción opcional con más detalle."},
				{Name: "reminders", Type: "array", Items: "string", Description: "Timestamps ISO8601 UTC para recordatorios. Ej: [\"2026-05-14T09:00:00Z\"]."},
			},
			Required: []string{"name", "deadline"},
		},
		{
			Name:        "update_task_status",
			Description: "Actualiza el estado de una tarea.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la tarea."},
				{Name: "status", Type: "string", Description: "Nuevo estado.", Enum: []string{"PENDING", "IN_PROGRESS", "DONE", "CANCELLED"}},
			},
			Required: []string{"id", "status"},
		},
		{
			Name:        "delete_task",
			Description: "Elimina una tarea permanentemente.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la tarea a eliminar."},
			},
			Required: []string{"id"},
		},
	}
}
