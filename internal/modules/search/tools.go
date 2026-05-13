package search

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	tools := []agentllm.ToolDef{
		{
			Name:        "fetch_url",
			Description: "Obtiene el contenido de una URL y lo convierte a texto plano. Útil para leer artículos, documentación, o páginas web completas.",
			Parameters: []agentllm.ParamDef{
				{Name: "url", Type: "string", Description: "URL completa (incluye https://)"},
			},
			Required: []string{"url"},
		},
	}
	if m.searcher != nil {
		tools = append([]agentllm.ToolDef{
			{
				Name:        "web_search",
				Description: "Busca en internet usando Google Search y retorna información actualizada sintetizada. Ej: web_search(\"clima Temuco hoy\"). La herramienta busca y presenta la respuesta automáticamente.",
				Parameters: []agentllm.ParamDef{
					{Name: "query", Type: "string", Description: "Término de búsqueda"},
				},
				Required: []string{"query"},
			},
		}, tools...)
	}
	return tools
}

func (m *Module) WriteTools() []agentllm.ToolDef { return nil }
