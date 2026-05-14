package search

import (
	"context"
	"fmt"

	ddgo "github.com/evgensoft/ddgo"
)

func (m *Module) search(ctx context.Context, query string) (map[string]any, error) {
	if query == "" {
		return nil, fmt.Errorf("web_search: query vacío")
	}

	results, err := ddgo.Query(query, 5)
	if err != nil {
		return nil, fmt.Errorf("web_search: %w", err)
	}

	type searchResult struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		URL     string `json:"url"`
	}

	out := make([]searchResult, 0, len(results))
	for _, r := range results {
		out = append(out, searchResult{
			Title:   r.Title,
			Snippet: r.Info,
			URL:     r.URL,
		})
	}

	return map[string]any{
		"ok":      true,
		"query":   query,
		"results": out,
		"total":   len(out),
	}, nil
}
