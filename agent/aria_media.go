// Package agent — aria_media.go maneja transcripción de audio (WhatsApp voice notes).
package agent

import "context"

// TranscribeAudio convierte una nota de voz (OGG) a texto via Whisper.
func (a *Aria) TranscribeAudio(ctx context.Context, data []byte) (string, error) {
	a.logger.Info("Transcribiendo audio", "bytes", len(data))
	return a.llm.Transcribe(ctx, data)
}
