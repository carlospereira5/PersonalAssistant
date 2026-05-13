package search

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "web_search":
		if m.searcher == nil {
			return nil, fmt.Errorf("web_search: no disponible (GEMINI_API_KEY no configurada)")
		}
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("web_search: query vacío")
		}
		return m.doSearch(ctx, query)

	case "fetch_url":
		url, _ := args["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("fetch_url: url vacía")
		}
		return m.fetch(ctx, url)
	}
	return nil, fmt.Errorf("search: herramienta desconocida %q", tool)
}

func (m *Module) doSearch(ctx context.Context, query string) (map[string]any, error) {
	result, err := m.searcher.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("web_search: %w", err)
	}

	return map[string]any{
		"ok":      true,
		"query":   query,
		"answer":  result.Answer,
		"sources": result.Sources,
		"queries": result.Queries,
	}, nil
}
