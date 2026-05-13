package llm

import (
	"context"
	"fmt"
	"iter"

	"google.golang.org/genai"
)

// geminiSession implementa Session usando genai.Chat.
//
// Comportamiento según configuración en NewSession (gemini.go):
//   - tools != nil: function declarations disponibles (tasks, scheduler CRUD)
//   - tools == nil: GoogleSearch Grounding activo (scheduler, texto plano)
type geminiSession struct {
	chat     *genai.Chat
	injected []*genai.Content
}

func (s *geminiSession) Send(ctx context.Context, message string) (string, []ToolCall, error) {
	if s.chat == nil {
		return "", nil, fmt.Errorf("gemini session: chat no inicializado")
	}

	parts := make([]*genai.Part, 0, 1+len(s.injected))
	for _, inj := range s.injected {
		for _, p := range inj.Parts {
			parts = append(parts, p)
		}
	}
	s.injected = nil
	parts = append(parts, &genai.Part{Text: message})

	resp, err := s.chat.Send(ctx, parts...)
	if err != nil {
		return "", nil, fmt.Errorf("gemini send: %w", err)
	}

	return parseGeminiResponse(resp)
}

func (s *geminiSession) SendToolResults(ctx context.Context, results []ToolResult) (string, []ToolCall, error) {
	if s.chat == nil {
		return "", nil, fmt.Errorf("gemini session: chat no inicializado")
	}

	parts := make([]*genai.Part, len(results))
	for i, r := range results {
		parts[i] = &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     r.Name,
				Response: r.Result,
			},
		}
	}

	resp, err := s.chat.Send(ctx, parts...)
	if err != nil {
		return "", nil, fmt.Errorf("gemini send tool results: %w", err)
	}

	return parseGeminiResponse(resp)
}

func (s *geminiSession) Stream(ctx context.Context, message string) (<-chan StreamEvent, error) {
	if s.chat == nil {
		ch := make(chan StreamEvent, 1)
		ch <- StreamEvent{Error: fmt.Errorf("gemini session: chat no inicializado")}
		close(ch)
		return ch, nil
	}

	ch := make(chan StreamEvent, 8)
	go func() {
		defer close(ch)

		parts := make([]*genai.Part, 0, 1+len(s.injected))
		for _, inj := range s.injected {
			for _, p := range inj.Parts {
				parts = append(parts, p)
			}
		}
		s.injected = nil
		parts = append(parts, &genai.Part{Text: message})

		stream := s.chat.SendStream(ctx, parts...)

		var fullText string
		var assistantToolCalls []genai.FunctionCall

		for chunk, err := range stream {
			if err != nil {
				ch <- StreamEvent{Error: fmt.Errorf("gemini stream: %w", err)}
				return
			}
			if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
				continue
			}
			for _, part := range chunk.Candidates[0].Content.Parts {
				if part == nil {
					continue
				}
				if part.Text != "" {
					fullText += part.Text
					ch <- StreamEvent{Text: part.Text}
				}
				if part.FunctionCall != nil {
					assistantToolCalls = append(assistantToolCalls, *part.FunctionCall)
				}
			}
		}

		if len(assistantToolCalls) > 0 {
			tcs := make([]ToolCall, len(assistantToolCalls))
			for i, fc := range assistantToolCalls {
				tcs[i] = ToolCall{Name: fc.Name, Args: fc.Args}
			}
			ch <- StreamEvent{ToolCalls: tcs, Done: true}
		} else {
			ch <- StreamEvent{Text: fullText, Done: true}
		}
	}()
	return ch, nil
}

func (s *geminiSession) StreamToolResults(ctx context.Context, results []ToolResult) (<-chan StreamEvent, error) {
	if s.chat == nil {
		ch := make(chan StreamEvent, 1)
		ch <- StreamEvent{Error: fmt.Errorf("gemini session: chat no inicializado")}
		close(ch)
		return ch, nil
	}

	ch := make(chan StreamEvent, 8)
	go func() {
		defer close(ch)

		parts := make([]*genai.Part, len(results))
		for i, r := range results {
			parts[i] = &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     r.Name,
					Response: r.Result,
				},
			}
		}

		stream := s.chat.SendStream(ctx, parts...)

		var fullText string
		var assistantToolCalls []genai.FunctionCall

		for chunk, err := range stream {
			if err != nil {
				ch <- StreamEvent{Error: fmt.Errorf("gemini stream: %w", err)}
				return
			}
			if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
				continue
			}
			for _, part := range chunk.Candidates[0].Content.Parts {
				if part == nil {
					continue
				}
				if part.Text != "" {
					fullText += part.Text
					ch <- StreamEvent{Text: part.Text}
				}
				if part.FunctionCall != nil {
					assistantToolCalls = append(assistantToolCalls, *part.FunctionCall)
				}
			}
		}

		if len(assistantToolCalls) > 0 {
			tcs := make([]ToolCall, len(assistantToolCalls))
			for i, fc := range assistantToolCalls {
				tcs[i] = ToolCall{Name: fc.Name, Args: fc.Args}
			}
			ch <- StreamEvent{ToolCalls: tcs, Done: true}
		} else {
			ch <- StreamEvent{Text: fullText, Done: true}
		}
	}()
	return ch, nil
}

func (s *geminiSession) InjectAssistantMessage(content string) {
	s.injected = append(s.injected, &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{{Text: content}},
	})
}

func parseGeminiResponse(resp *genai.GenerateContentResponse) (string, []ToolCall, error) {
	if len(resp.Candidates) == 0 {
		return "", nil, fmt.Errorf("gemini: 0 candidates en respuesta")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return "", nil, fmt.Errorf("gemini: candidate sin content")
	}

	var text string
	var funcCalls []genai.FunctionCall

	for _, part := range candidate.Content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			text += part.Text
		}
		if part.FunctionCall != nil {
			funcCalls = append(funcCalls, *part.FunctionCall)
		}
	}

	if len(funcCalls) == 0 {
		return text, nil, nil
	}

	tcs := make([]ToolCall, len(funcCalls))
	for i, fc := range funcCalls {
		tcs[i] = ToolCall{Name: fc.Name, Args: fc.Args}
	}

	return text, tcs, nil
}

// compile-time check: iter.Seq2 is used by genai's streaming API
var _ iter.Seq2[*genai.GenerateContentResponse, error]
