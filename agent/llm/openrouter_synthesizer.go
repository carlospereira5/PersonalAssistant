package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenRouterSynthesizer implementa Synthesizer usando openai/gpt-audio-mini vía OpenRouter.
// Usa chat completions con stream=true y modality "audio".
type OpenRouterSynthesizer struct {
	apiKey string
	model  string
	voice  string
	client *http.Client
}

func NewOpenRouterSynthesizer(apiKey, model, voice string) *OpenRouterSynthesizer {
	if model == "" {
		model = "openai/gpt-audio-mini"
	}
	if voice == "" {
		voice = "coral"
	}
	return &OpenRouterSynthesizer{
		apiKey: apiKey,
		model:  model,
		voice:  voice,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *OpenRouterSynthesizer) Synthesize(ctx context.Context, text string) ([]byte, error) {
	audioData, err := s.callStreamingAudio(ctx, text)
	if err != nil {
		return nil, err
	}
	return pcm16ToOGGOpus(ctx, audioData)
}

func (s *OpenRouterSynthesizer) callStreamingAudio(ctx context.Context, text string) ([]byte, error) {
	body := map[string]any{
		"model":      s.model,
		"modalities": []string{"text", "audio"},
		"stream":     true,
		"audio": map[string]string{
			"voice":  s.voice,
			"format": "pcm16",
		},
		"messages": []map[string]string{
			{"role": "user", "content": text},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openrouter audio: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("openrouter audio: crear request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter audio: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter audio: status %d: %s", resp.StatusCode, body)
	}

	var audioB64 strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Audio struct {
						Data string `json:"data"`
					} `json:"audio"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			audioB64.WriteString(chunk.Choices[0].Delta.Audio.Data)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("openrouter audio: leyendo stream: %w", err)
	}
	if audioB64.Len() == 0 {
		return nil, fmt.Errorf("openrouter audio: no se recibió audio en el stream")
	}

	decoded, err := base64.StdEncoding.DecodeString(audioB64.String())
	if err != nil {
		return nil, fmt.Errorf("openrouter audio: decode base64: %w", err)
	}
	return decoded, nil
}
