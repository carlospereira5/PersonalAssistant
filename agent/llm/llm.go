// Package llm define las interfaces y tipos para comunicarse con modelos de lenguaje.
// Contiene las implementaciones concretas (Gemini, OpenAI/Groq) y la gestión
// del ciclo de vida de sesiones (SessionManager, retrySession).
package llm

import "context"

// LLM crea sesiones de conversación, transcribe audio y analiza imágenes.
type LLM interface {
	NewSession(ctx context.Context, systemPrompt string, tools []ToolDef) (Session, error)
	Transcribe(ctx context.Context, audioData []byte) (string, error)
	// AnalyzeImage envía una imagen al modelo de visión y retorna texto estructurado.
	// mimeType es el tipo MIME de la imagen (ej: "image/jpeg").
	AnalyzeImage(ctx context.Context, imageData []byte, mimeType, prompt string) (string, error)
}

// Session es una conversación stateful con un LLM.
type Session interface {
	Send(ctx context.Context, message string) (text string, calls []ToolCall, err error)
	SendToolResults(ctx context.Context, results []ToolResult) (text string, calls []ToolCall, err error)
	
	// Stream inicia un stream de la respuesta para el mensaje dado.
	Stream(ctx context.Context, message string) (<-chan StreamEvent, error)
	// StreamToolResults inicia un stream tras recibir resultados de herramientas.
	StreamToolResults(ctx context.Context, results []ToolResult) (<-chan StreamEvent, error)

	// InjectAssistantMessage inserta un turno del asistente en el historial de la
	// sesión sin llamar al LLM. Úsalo para dar contexto previo generado fuera de
	// Chat (ej: respuesta de ProcessInvoice en el admin TUI).
	InjectAssistantMessage(content string)
}
