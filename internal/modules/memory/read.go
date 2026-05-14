package memory

import (
	"context"
	"fmt"
)

func (m *Module) Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "get_fact":
		return m.getFact(ctx, args)
	case "search_memory":
		return m.searchMemory(ctx, args)
	}
	return nil, fmt.Errorf("memory: herramienta desconocida %q", tool)
}

func (m *Module) getFact(ctx context.Context, args map[string]any) (map[string]any, error) {
	key, _ := args["key"].(string)
	if key == "" {
		return nil, fmt.Errorf("get_fact: key requerida")
	}

	fact, err := m.store.GetFact(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get_fact: %w", err)
	}
	if fact == nil {
		return map[string]any{"found": false, "key": key}, nil
	}

	result := map[string]any{
		"found":      true,
		"key":        fact.Key,
		"value":      fact.Value,
		"created_at": fact.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if fact.ExpiresAt != nil {
		result["expires_at"] = fact.ExpiresAt.Format("2006-01-02 15:04:05")
	}
	return result, nil
}

func (m *Module) searchMemory(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("search_memory: query requerida")
	}

	facts, err := m.store.SearchFacts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search_memory: %w", err)
	}

	results := make([]map[string]any, 0, len(facts))
	for _, f := range facts {
		entry := map[string]any{
			"key":        f.Key,
			"value":      f.Value,
			"created_at": f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if f.ExpiresAt != nil {
			entry["expires_at"] = f.ExpiresAt.Format("2006-01-02 15:04:05")
		}
		results = append(results, entry)
	}

	return map[string]any{"results": results, "total": len(results)}, nil
}
