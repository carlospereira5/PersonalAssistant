package llm

import (
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

const googleTTSEndpoint = "https://texttospeech.googleapis.com/v1/text:synthesize"

// GoogleSynthesizer implementa Synthesizer usando Google Cloud Text-to-Speech.
// Tier gratuito: 1M caracteres/mes con voces Neural2.
// Sin SDK adicional — REST API pura con net/http.
//
// Setup:
//  1. Habilitar "Cloud Text-to-Speech API" en console.cloud.google.com
//  2. Crear API Key en "APIs & Services → Credentials"
//  3. Agregar GOOGLE_TTS_KEY a Infisical
type GoogleSynthesizer struct {
	apiKey string
	voice  string // ej: "es-US-Neural2-A" (default) o "es-US-Chirp3-HD-Aoede"
	client *http.Client
}

// NewGoogleSynthesizer crea un sintetizador usando Google Cloud TTS.
// voice: nombre de voz de Google TTS, ej "es-US-Neural2-A".
// Voces disponibles: https://cloud.google.com/text-to-speech/docs/voices
func NewGoogleSynthesizer(apiKey, voice string) *GoogleSynthesizer {
	return &GoogleSynthesizer{
		apiKey: apiKey,
		voice:  voice,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Synthesize convierte texto a audio OGG/Opus.
// Llama a Google TTS REST API → obtiene MP3 → convierte a OGG/Opus con ffmpeg.
func (s *GoogleSynthesizer) Synthesize(ctx context.Context, text string) ([]byte, error) {
	mp3Data, err := s.callTTS(ctx, text)
	if err != nil {
		return nil, err
	}
	return audioToOGGOpus(ctx, mp3Data)
}

func (s *GoogleSynthesizer) callTTS(ctx context.Context, text string) ([]byte, error) {
	// Derivar languageCode del nombre de voz: "es-US-Neural2-A" → "es-US"
	parts := strings.SplitN(s.voice, "-", 3)
	langCode := s.voice
	if len(parts) >= 2 {
		langCode = parts[0] + "-" + parts[1]
	}

	body := struct {
		Input struct {
			Text string `json:"text"`
		} `json:"input"`
		Voice struct {
			LanguageCode string `json:"languageCode"`
			Name         string `json:"name"`
		} `json:"voice"`
		AudioConfig struct {
			AudioEncoding string `json:"audioEncoding"`
		} `json:"audioConfig"`
	}{}
	body.Input.Text = text
	body.Voice.LanguageCode = langCode
	body.Voice.Name = s.voice
	body.AudioConfig.AudioEncoding = "MP3"

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("google tts: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		googleTTSEndpoint+"?key="+s.apiKey,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("google tts: crear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google tts: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google tts: leer respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google tts: status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		AudioContent string `json:"audioContent"` // base64-encoded MP3
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("google tts: parse respuesta: %w", err)
	}

	mp3Data, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("google tts: decode base64: %w", err)
	}

	return mp3Data, nil
}
