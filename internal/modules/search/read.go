package search

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "web_search":
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("web_search: query vacío")
		}
		return m.search(ctx, query)

	case "fetch_url":
		url, _ := args["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("fetch_url: url vacía")
		}
		return m.fetch(ctx, url)
	}
	return nil, fmt.Errorf("search: herramienta desconocida %q", tool)
}
