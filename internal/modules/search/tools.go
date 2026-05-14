package search

import agentllm "github.com/carlospereira5/PersonalAssistant/agent/llm"

func (m *Module) ReadTools() []agentllm.ToolDef {
	return []agentllm.ToolDef{
		// web_search es un server tool de OpenRouter. El modelo decide cuándo
		// buscar, y OpenRouter ejecuta la búsqueda server-side usando Exa.
		// La respuesta vuelve con los resultados de búsqueda incorporados
		// automáticamente — no pasa por nuestro dispatcher.
		{ServerTool: "openrouter:web_search"},
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
