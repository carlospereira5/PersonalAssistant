package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/sashabaranov/go-openai"
)

// Synthesizer convierte texto a audio OGG/Opus para notas de voz de WhatsApp.
// La implementación concreta decide el proveedor TTS.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}

// GroqSynthesizer implementa Synthesizer usando Groq PlayAI TTS.
// Usa voces Spanish-native (ej: "Celeste-PlayAI") para sonar natural en español.
// Retorna bytes OGG/Opus listos para enviar como nota de voz PTT en WhatsApp.
//
// Requiere ffmpeg instalado en el sistema (Ubuntu: apt install ffmpeg,
// Termux: pkg install ffmpeg).
type GroqSynthesizer struct {
	client *openai.Client
	voice  string
}

// NewGroqSynthesizer crea un sintetizador usando la API TTS de Groq (Orpheus v1).
// voice: nombre de la voz Orpheus. Femeninas: "autumn", "diana", "hannah". Masculinas: "austin", "daniel", "troy".
// Voces disponibles: https://console.groq.com/docs/text-to-speech
func NewGroqSynthesizer(client *openai.Client, voice string) *GroqSynthesizer {
	if voice == "" {
		voice = "diana"
	}
	return &GroqSynthesizer{client: client, voice: voice}
}

// Synthesize convierte texto a audio OGG/Opus.
// Internamente llama a Groq Orpheus TTS → obtiene MP3 → convierte a OGG/Opus con ffmpeg.
func (s *GroqSynthesizer) Synthesize(ctx context.Context, text string) ([]byte, error) {
	resp, err := s.client.CreateSpeech(ctx, openai.CreateSpeechRequest{
		Model:          openai.SpeechModel("canopylabs/orpheus-v1-english"),
		Input:          text,
		Voice:          openai.SpeechVoice(s.voice),
		ResponseFormat: openai.SpeechResponseFormatWav,
	})
	if err != nil {
		return nil, fmt.Errorf("groq tts: %w", err)
	}
	defer resp.Close()

	wavData, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("groq tts: leyendo respuesta: %w", err)
	}

	return audioToOGGOpus(ctx, wavData)
}

// pcm16ToOGGOpus convierte PCM16 raw (s16le, 24kHz, mono) a OGG/Opus.
// PCM raw no tiene header, por eso ffmpeg necesita los parámetros de formato explícitos.
// OpenAI devuelve PCM16 a 24000 Hz, mono, cuando format="pcm16".
func pcm16ToOGGOpus(ctx context.Context, data []byte) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "s16le", "-ar", "24000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "libopus",
		"-b:a", "32k",
		"-f", "ogg",
		"pipe:1",
		"-loglevel", "error",
		"-y",
	)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg pcm16→ogg: %w (stderr: %s)", err, stderr.String())
	}
	return out, nil
}

// audioToOGGOpus convierte audio (MP3, WAV u otro formato soportado por ffmpeg) a OGG/Opus.
// WhatsApp exige OGG/Opus para notas de voz PTT.
func audioToOGGOpus(ctx context.Context, data []byte) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", "pipe:0", // ffmpeg auto-detecta el formato de entrada
		"-c:a", "libopus",
		"-b:a", "32k",
		"-f", "ogg",
		"pipe:1",
		"-loglevel", "error",
		"-y",
	)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg→ogg: %w (stderr: %s)", err, stderr.String())
	}
	return out, nil
}
