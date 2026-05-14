package search

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "web_search":
		// web_search es un server tool de OpenRouter — se maneja server-side.
		// El modelo lo invoca y OpenRouter ejecuta la búsqueda, devolviendo
		// los resultados incorporados en la respuesta. Nunca llega acá.
		return nil, fmt.Errorf("web_search: server tool handled by OpenRouter")

	case "fetch_url":
		url, _ := args["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("fetch_url: url vacía")
		}
		return m.fetch(ctx, url)
	}
	return nil, fmt.Errorf("search: herramienta desconocida %q", tool)
}
