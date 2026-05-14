package memory

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "get_fact",
			Description: "Recupera un hecho guardado por su clave exacta. Si no existe o expiró, retorna null.",
			Parameters: []agentllm.ParamDef{
				{Name: "key", Type: "string", Description: "Clave del hecho a recuperar. Usá notación con prefijos: \"user/nombre\", \"user/timezone\", \"task/preferencia\"."},
			},
			Required: []string{"key"},
		},
		{
			Name:        "search_memory",
			Description: "Busca en todos los hechos guardados usando búsqueda por texto completo. Retorna hasta 20 resultados ordenados por relevancia.",
			Parameters: []agentllm.ParamDef{
				{Name: "query", Type: "string", Description: "Términos de búsqueda. Ej: \"preferencias usuario\", \"configuración horario\", \"gustos musicales\"."},
			},
			Required: []string{"query"},
		},
	}
}

func (m *Module) WriteTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "save_fact",
			Description: "Guarda un hecho en la memoria persistente del agente. El agente recuerda esta información entre sesiones. Si ya existe un hecho con la misma key, se actualiza. Usalo para recordar preferencias del usuario, datos personales, decisiones, configuraciones.",
			Parameters: []agentllm.ParamDef{
				{Name: "key", Type: "string", Description: "Clave única del hecho. Usá notación con prefijos para organizar: \"user/preferencia\", \"config/timezone\", \"task/default_priority\"."},
				{Name: "value", Type: "string", Description: "Valor del hecho. Texto libre con la información a recordar."},
				{Name: "ttl", Type: "integer", Description: "Tiempo de vida en segundos (opcional). Si se omite, el hecho no expira. Ej: 86400 = 1 día, 604800 = 1 semana."},
			},
			Required: []string{"key", "value"},
		},
	}
}
