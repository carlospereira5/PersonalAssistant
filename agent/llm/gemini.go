package llm

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GeminiLLM implementa LLM usando la API nativa de Google Gemini.
//
// Comportamiento de NewSession según tools:
//   - tools != nil (chat): sesión con function declarations (tasks, scheduler).
//     web_search se maneja vía tool call → GeminiSearch REST API.
//   - tools == nil (scheduler): sesión con GoogleSearch Grounding.
//     El modelo busca en Google automáticamente.
//
// Para Transcribe y AnalyzeImage delega a openAILLM como fallback.
type GeminiLLM struct {
	client    *genai.Client
	model     string
	openAILLM *OpenAILLM
}

// NewGeminiLLM crea un LLM Gemini. Limpia el model name automáticamente
// (saca prefijos "google/", "models/", "gemini/" para API nativa).
func NewGeminiLLM(client *genai.Client, modelName string, openAILLM *OpenAILLM) *GeminiLLM {
	clean := strings.TrimPrefix(modelName, "google/")
	clean = strings.TrimPrefix(clean, "models/")
	clean = strings.TrimPrefix(clean, "gemini/")
	return &GeminiLLM{
		client:    client,
		model:     clean,
		openAILLM: openAILLM,
	}
}

// NewSession crea una sesión Gemini.
// - tools con entries: function declarations (chat). Sin GoogleSearch.
// - tools vacío/nil: GoogleSearch Grounding (scheduler, texto plano).
func (g *GeminiLLM) NewSession(ctx context.Context, systemPrompt string, tools []ToolDef) (Session, error) {
	config := &genai.GenerateContentConfig{}

	if systemPrompt != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		}
	}

	if len(tools) == 0 {
		// Scheduler: solo GoogleSearch (Gemini 2.5 no permite combinarlos)
		config.Tools = []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		}
	} else {
		// Chat: solo function declarations
		genaiTools := make([]*genai.Tool, len(tools))
		for i, t := range tools {
			fd := &genai.FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
			}

			if len(t.Parameters) > 0 {
				schema := &genai.Schema{
					Type:       genai.TypeObject,
					Properties: make(map[string]*genai.Schema),
				}

				for _, p := range t.Parameters {
					paramSchema := &genai.Schema{
						Description: p.Description,
						Type:        toGenaiType(p.Type),
					}

					if p.Type == "array" && p.Items != "" {
						paramSchema.Items = &genai.Schema{
							Type: toGenaiType(p.Items),
						}
					}

					if len(p.Enum) > 0 {
						paramSchema.Enum = p.Enum
					}

					schema.Properties[p.Name] = paramSchema
				}

				if len(t.Required) > 0 {
					schema.Required = t.Required
				}

				fd.Parameters = schema
			}

			genaiTools[i] = &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{fd},
			}
		}

		config.Tools = genaiTools
	}

	chat, err := g.client.Chats.Create(ctx, g.model, config, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini new session: %w", err)
	}

	return &geminiSession{chat: chat}, nil
}

func (g *GeminiLLM) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if g.openAILLM != nil {
		return g.openAILLM.Transcribe(ctx, audioData)
	}
	return "", fmt.Errorf("gemini: transcripción no disponible (sin fallback OpenAI)")
}

func (g *GeminiLLM) AnalyzeImage(ctx context.Context, imageData []byte, mimeType, prompt string) (string, error) {
	if g.openAILLM != nil {
		return g.openAILLM.AnalyzeImage(ctx, imageData, mimeType, prompt)
	}
	return "", fmt.Errorf("gemini: análisis de imagen no disponible (sin fallback OpenAI)")
}

func toGenaiType(t string) genai.Type {
	switch strings.ToLower(t) {
	case "string":
		return genai.TypeString
	case "integer":
		return genai.TypeInteger
	case "number":
		return genai.TypeNumber
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	default:
		return genai.TypeString
	}
}
