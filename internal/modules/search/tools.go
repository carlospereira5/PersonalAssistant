package search

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		{
			Name:        "web_search",
			Description: "Busca en internet usando DuckDuckGo. Útil para obtener información actualizada, noticias, o cualquier consulta que requiera datos frescos.",
			Parameters: []agentllm.ParamDef{
				{Name: "query", Type: "string", Description: "Término de búsqueda"},
			},
			Required: []string{"query"},
		},
		{
			Name:        "fetch_url",
			Description: "Obtiene el contenido de una URL y lo convierte a texto plano. Útil para leer artículos, documentación, o páginas web completas.",
			Parameters: []agentllm.ParamDef{
				{Name: "url", Type: "string", Description: "URL completa (incluye https://)"},
			},
			Required: []string{"url"},
		},
	}
}

func (m *Module) WriteTools() []agentllm.ToolDef { return nil }
