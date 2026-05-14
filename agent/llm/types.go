package llm

// ToolDef define una herramienta que el LLM puede invocar.
type ToolDef struct {
	Name        string
	Description string
	Parameters  []ParamDef
	Required    []string

	// ServerTool, si no está vacío, indica que este tool es un server tool
	// (ej: "openrouter:web_search"). En ese caso, Name, Description, Parameters
	// y Required se ignoran, y el tool se envía con el type especificado.
	// Server tools son ejecutados por el proveedor (OpenRouter) server-side,
	// no por nuestro código.
	ServerTool string
}

// ParamDef define un parámetro de una herramienta.
type ParamDef struct {
	Name        string
	Type        string // "string", "integer", "number", "boolean", "array"
	Description string
	Enum        []string
	Items       string // tipo de los elementos cuando Type=="array", e.g. "string"
}

// ToolCall representa una invocación de herramienta pedida por el LLM.
type ToolCall struct {
	Name string
	Args map[string]any
}

// ToolResult es el resultado de ejecutar una herramienta.
type ToolResult struct {
	Name   string
	Result map[string]any
}

// StreamEvent representa un fragmento de la respuesta del LLM o un evento de herramienta.
type StreamEvent struct {
	Text      string     // Fragmento de texto
	ToolCalls []ToolCall // Invocación de herramienta (usualmente al final del stream)
	Done      bool       // Indica fin del stream
	Error     error      // Error ocurrido durante el streaming
}
