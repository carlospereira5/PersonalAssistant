package scheduler

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "list_routines",
			Description: "Retorna todas las rutinas programadas con su estado, expresión cron, preview del prompt y última ejecución.",
		},
	}
}

func (m *Module) WriteTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "create_routine",
			Description: "Crea una nueva rutina programada. La expresión cron debe tener formato \"minuto hora día-del-mes mes día-de-la-semana\". El prompt es el texto que el asistente ejecutará cuando la rutina se active.",
			Parameters: []agentllm.ParamDef{
				{Name: "cron", Type: "string", Description: "Expresión cron de 5 campos: minuto hora día-del-mes mes día-de-la-semana. Ej: \"0 8 * * *\" = todos los días a las 08:00."},
				{Name: "prompt", Type: "string", Description: "Texto que el asistente ejecutará cuando la rutina se active."},
			},
			Required: []string{"cron", "prompt"},
		},
		{
			Name:        "delete_routine",
			Description: "Elimina una rutina programada permanentemente.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la rutina a eliminar."},
			},
			Required: []string{"id"},
		},
		{
			Name:        "pause_routine",
			Description: "Pausa una rutina activa. No se ejecutará hasta que se reanude.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la rutina a pausar."},
			},
			Required: []string{"id"},
		},
		{
			Name:        "resume_routine",
			Description: "Reanuda una rutina pausada. Volverá a ejecutarse según su programación cron.",
			Parameters: []agentllm.ParamDef{
				{Name: "id", Type: "integer", Description: "ID de la rutina a reanudar."},
			},
			Required: []string{"id"},
		},
	}
}
