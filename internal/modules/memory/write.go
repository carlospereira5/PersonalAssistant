package memory

import (
	"context"
	"fmt"
)

func (m *Module) Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error) {
	switch tool {
	case "save_fact":
		return m.saveFact(ctx, args)
	}
	return nil, fmt.Errorf("memory: herramienta desconocida %q", tool)
}

func (m *Module) saveFact(ctx context.Context, args map[string]any) (map[string]any, error) {
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	if key == "" {
		return nil, fmt.Errorf("save_fact: key requerida")
	}
	if value == "" {
		return nil, fmt.Errorf("save_fact: value requerido")
	}

	ttl := 0
	if ttlVal, ok := args["ttl"]; ok {
		switch v := ttlVal.(type) {
		case float64:
			ttl = int(v)
		case int:
			ttl = v
		}
	}

	if err := m.store.SaveFact(ctx, key, value, ttl); err != nil {
		return nil, fmt.Errorf("save_fact: %w", err)
	}

	m.logger.Info("Fact guardado", "key", key, "ttl", ttl)
	result := map[string]any{"ok": true, "key": key, "value": value}
	if ttl > 0 {
		result["ttl_seconds"] = ttl
	}
	return result, nil
}
