// Package search — Gemini Search Grounding vía REST API directa.
//
// Usa la Gemini API con google_search tool para obtener respuestas con
// información actualizada de internet. NO usa el SDK genai para evitar
// conflictos con function declarations en Gemini 2.5.
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

// SearchClient llama a Gemini con Google Search Grounding.
type SearchClient struct {
	apiKey string
	model  string
	client *http.Client
}

// NewSearchClient crea un cliente de búsqueda Google Grounding.
func NewSearchClient(apiKey, model string) *SearchClient {
	return &SearchClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SearchResult contiene la respuesta con grounding.
type SearchResult struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources,omitempty"`
	Queries []string `json:"queries,omitempty"`
}

// Search envía un query a Gemini con GoogleSearch y retorna respuesta sintetizada.
func (s *SearchClient) Search(ctx context.Context, query string) (*SearchResult, error) {
	body := newGeminiRequestBody(query)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini search: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf(geminiEndpoint, s.model),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("gemini search: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini search: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini search: read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini search: status %d: %s", resp.StatusCode, string(respBody))
	}

	return parseResponse(respBody)
}

type geminiRequestBody struct {
	Contents []geminiContent `json:"contents"`
	Tools    []geminiTool    `json:"tools"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiTool struct {
	GoogleSearch *geminiGoogleSearch `json:"googleSearch,omitempty"`
}

type geminiGoogleSearch struct{}

func newGeminiRequestBody(query string) geminiRequestBody {
	return geminiRequestBody{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: query}}},
		},
		Tools: []geminiTool{
			{GoogleSearch: &geminiGoogleSearch{}},
		},
	}
}

type geminiResponse struct {
	Candidates []struct {
		Content           *geminiContent     `json:"content,omitempty"`
		GroundingMetadata *groundingMetadata `json:"groundingMetadata,omitempty"`
	} `json:"candidates"`
}

type groundingMetadata struct {
	WebSearchQueries []string         `json:"webSearchQueries,omitempty"`
	GroundingChunks  []groundingChunk `json:"groundingChunks,omitempty"`
}

type groundingChunk struct {
	Web struct {
		URI   string `json:"uri,omitempty"`
		Title string `json:"title,omitempty"`
	} `json:"web,omitempty"`
}

func parseResponse(body []byte) (*SearchResult, error) {
	var resp geminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("gemini search: parse: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini search: 0 candidates")
	}

	candidate := resp.Candidates[0]
	result := &SearchResult{}

	if candidate.Content != nil {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				result.Answer = strings.TrimSpace(part.Text)
				break
			}
		}
	}

	if meta := candidate.GroundingMetadata; meta != nil {
		result.Queries = meta.WebSearchQueries
		for _, chunk := range meta.GroundingChunks {
			if chunk.Web.URI != "" {
				result.Sources = append(result.Sources, chunk.Web.URI)
			}
		}
	}

	if result.Answer == "" {
		return nil, fmt.Errorf("gemini search: respuesta vacía")
	}

	return result, nil
}
