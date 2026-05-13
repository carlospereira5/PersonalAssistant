package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	ddgo "github.com/evgensoft/ddgo"
	"github.com/k3a/html2text"
)

func (m *Module) fetch(ctx context.Context, url string) (map[string]any, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("fetch_url: URL debe comenzar con http:// o https://")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch_url: %w", err)
	}
	req.Header.Set("User-Agent", ddgo.DefaultUserAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch_url: %w", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "text/html") && !strings.HasPrefix(ct, "text/plain") {
		return nil, fmt.Errorf("fetch_url: Content-Type %q no soportado (solo text/html, text/plain)", ct)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 200_000))
	if err != nil {
		return nil, fmt.Errorf("fetch_url: leyendo body: %w", err)
	}

	// Primero convertir HTML a texto plano, LUEGO truncar.
	// Truncar el HTML antes rompe la estructura y html2text devuelve vacío.
	text := html2text.HTML2Text(string(body))

	truncated := false
	if len(text) > 8192 {
		text = text[:8192]
		truncated = true
	}

	return map[string]any{
		"ok":           true,
		"url":          url,
		"content_type": ct,
		"char_count":   len(text),
		"truncated":    truncated,
		"text":         text,
	}, nil
}
