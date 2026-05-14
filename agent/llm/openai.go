package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAILLM implementa LLM usando el SDK compatible con OpenAI/Groq.
type OpenAILLM struct {
	client        *openai.Client
	whisperClient *openai.Client // cliente exclusivo para Whisper (puede ser nil → usa client)
	model         string
	visionModel   string
}

// NewOpenAILLM crea un LLM compatible con OpenAI. whisperClient puede ser nil;
// en ese caso Transcribe usa el mismo client principal.
func NewOpenAILLM(client *openai.Client, model, visionModel string, whisperClient *openai.Client) *OpenAILLM {
	return &OpenAILLM{client: client, whisperClient: whisperClient, model: model, visionModel: visionModel}
}

// Transcribe envía el audio OGG de WhatsApp al endpoint Whisper.
// Usa whisperClient si está configurado (ej: Groq cuando el cliente principal es OpenRouter).
func (o *OpenAILLM) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	c := o.client
	if o.whisperClient != nil {
		c = o.whisperClient
	}
	req := openai.AudioRequest{
		Model:    "whisper-large-v3-turbo",
		Reader:   bytes.NewReader(audioData),
		FilePath: "voice_note.ogg",
		Format:   openai.AudioResponseFormatText,
	}
	resp, err := c.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai transcription: %w", err)
	}
	return resp.Text, nil
}

// AnalyzeImage envía una imagen al endpoint de visión compatible con OpenAI.
// El modelo se configura via OPENAI_VISION_MODEL (default: meta-llama/llama-4-scout-17b-16e-instruct).
func (o *OpenAILLM) AnalyzeImage(ctx context.Context, imageData []byte, mimeType, prompt string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imageData))
	req := openai.ChatCompletionRequest{
		Model: o.visionModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type:     openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{URL: dataURL},
					},
					{Type: openai.ChatMessagePartTypeText, Text: prompt},
				},
			},
		},
		Temperature: 0.1,
	}
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai vision: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai vision: 0 choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func (o *OpenAILLM) NewSession(_ context.Context, systemPrompt string, tools []ToolDef) (Session, error) {
	oTools := make([]openai.Tool, len(tools))
	for i, t := range tools {
		// Server tools (ej: openrouter:web_search) — el proveedor las maneja server-side.
		if t.ServerTool != "" {
			oTools[i] = openai.Tool{
				Type: openai.ToolType(t.ServerTool),
			}
			continue
		}

		props := make(map[string]any, len(t.Parameters))
		for _, p := range t.Parameters {
			var prop map[string]any
			if p.Items != "" {
				prop = map[string]any{"type": "array", "items": map[string]any{"type": p.Items}, "description": p.Description}
			} else {
				prop = map[string]any{"type": p.Type, "description": p.Description}
				if len(p.Enum) > 0 {
					prop["enum"] = p.Enum
				}
			}
			props[p.Name] = prop
		}
		schema := map[string]any{"type": "object", "properties": props}
		if len(t.Required) > 0 {
			schema["required"] = t.Required
		}
		schemaBytes, err := json.Marshal(schema)
		if err != nil {
			return nil, fmt.Errorf("marshaling tool schema: %w", err)
		}
		oTools[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(schemaBytes),
			},
		}
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
	}
	return &openAISession{
		client:          o.client,
		model:           o.model,
		messages:        messages,
		tools:           oTools,
		lastToolCallIDs: make(map[string]string),
	}, nil
}

type openAISession struct {
	client          *openai.Client
	model           string
	messages        []openai.ChatCompletionMessage
	tools           []openai.Tool
	lastToolCallIDs map[string]string
}

func (s *openAISession) Send(ctx context.Context, message string) (string, []ToolCall, error) {
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})
	return s.do(ctx)
}

func (s *openAISession) Stream(ctx context.Context, message string) (<-chan StreamEvent, error) {
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})
	return s.stream(ctx), nil
}

func (s *openAISession) SendToolResults(ctx context.Context, results []ToolResult) (string, []ToolCall, error) {
	for _, r := range results {
		contentBytes, err := json.Marshal(r.Result)
		if err != nil {
			return "", nil, fmt.Errorf("marshaling tool result: %w", err)
		}
		s.messages = append(s.messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    string(contentBytes),
			Name:       r.Name,
			ToolCallID: s.lastToolCallIDs[r.Name],
		})
	}
	return s.do(ctx)
}

func (s *openAISession) StreamToolResults(ctx context.Context, results []ToolResult) (<-chan StreamEvent, error) {
	for _, r := range results {
		contentBytes, err := json.Marshal(r.Result)
		if err != nil {
			return nil, fmt.Errorf("marshaling tool result: %w", err)
		}
		s.messages = append(s.messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    string(contentBytes),
			Name:       r.Name,
			ToolCallID: s.lastToolCallIDs[r.Name],
		})
	}
	return s.stream(ctx), nil
}

func (s *openAISession) InjectAssistantMessage(content string) {
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	})
}

func (s *openAISession) do(ctx context.Context) (string, []ToolCall, error) {
	req := openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    s.messages,
		Tools:       s.tools,
		Temperature: 0.3,
	}
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("openai chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("openai devolvió 0 choices")
	}

	msg := resp.Choices[0].Message
	s.messages = append(s.messages, msg)

	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			s.lastToolCallIDs[tc.Function.Name] = tc.ID
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return "", nil, fmt.Errorf("parsing tool args: %w", err)
			}
			toolCalls[i] = ToolCall{Name: tc.Function.Name, Args: args}
		}
		return "", toolCalls, nil
	}
	return msg.Content, nil, nil
}

func (s *openAISession) stream(ctx context.Context) <-chan StreamEvent {
	ch := make(chan StreamEvent, 1)
	req := openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    s.messages,
		Tools:       s.tools,
		Temperature: 0.3,
		Stream:      true,
	}

	go func() {
		defer close(ch)
		stream, err := s.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("openai stream error: %w", err)}
			return
		}
		defer stream.Close()

		var fullContent strings.Builder
		var assistantToolCalls []openai.ToolCall

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				ch <- StreamEvent{Error: fmt.Errorf("stream read error: %w", err)}
				return
			}

			if len(resp.Choices) == 0 {
				continue
			}

			delta := resp.Choices[0].Delta

			// Acumular contenido de texto
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				ch <- StreamEvent{Text: delta.Content}
			}

			// Acumular tool calls (en stream vienen por fragmentos)
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					if tc.Index != nil {
						idx := *tc.Index
						for len(assistantToolCalls) <= idx {
							assistantToolCalls = append(assistantToolCalls, openai.ToolCall{})
						}
						if tc.ID != "" {
							assistantToolCalls[idx].ID = tc.ID
						}
						if tc.Function.Name != "" {
							assistantToolCalls[idx].Function.Name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							assistantToolCalls[idx].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}
		}

		// Al finalizar, guardamos el mensaje del asistente en el historial
		msg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fullContent.String(),
		}

		var toolCalls []ToolCall
		if len(assistantToolCalls) > 0 {
			msg.ToolCalls = assistantToolCalls
			toolCalls = make([]ToolCall, len(assistantToolCalls))
			for i, tc := range assistantToolCalls {
				s.lastToolCallIDs[tc.Function.Name] = tc.ID
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					// Si fallamos parseando args, enviamos el error por el canal
					ch <- StreamEvent{Error: fmt.Errorf("parsing tool args: %w", err)}
					return
				}
				toolCalls[i] = ToolCall{Name: tc.Function.Name, Args: args}
			}
		}

		s.messages = append(s.messages, msg)
		ch <- StreamEvent{ToolCalls: toolCalls, Done: true}
	}()

	return ch
}
