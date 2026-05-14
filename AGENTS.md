# Personal Assistant — Agent Architecture Reference

> Este documento define los principios arquitectónicos, el modelo de desarrollo y el código de conducta del agente. Funciona como constitución del proyecto: toda decisión de diseño debe poder rastrearse hasta uno de estos fundamentos.

---

## Tabla de Contenidos

1. [Filosofía Arquitectónica](#1-filosofía-arquitectónica)
2. [Arquitectura de Referencia: 5 Capas](#2-arquitectura-de-referencia-5-capas)
3. [Modelo de Capacidades: Módulos Gen2](#3-modelo-de-capacidades-módulos-gen2)
4. [Lifecycle Hooks](#4-lifecycle-hooks)
5. [Arquitectura de Memoria](#5-arquitectura-de-memoria)
6. [Código de Conducta del Agente](#6-código-de-conducta-del-agente)
7. [Flujo de Trabajo por Sesión](#7-flujo-de-trabajo-por-sesión)
8. [Evolución y Gobernanza](#8-evolución-y-gobernanza)

---

## 1. Filosofía Arquitectónica

### 1.1 Principios Fundamentales

| Principio | Enunciado | Implicancia |
|---|---|---|
| **Separación de Capas** | Cada capa tiene una responsabilidad única y no conoce a las demás | `agent/` es solo orquestación; no contiene lógica de negocio |
| **Capacidades como Módulos** | Toda habilidad del agente es un módulo autocontenido | Cada módulo vive en `internal/modules/<name>/` y es independiente |
| **Cero Acoplamiento** | Los módulos se comunican solo a través de interfaces en `agent/port.go` | No hay imports entre módulos; solo dependen de `agent.PortDeps` |
| **Herencia por Composición** | Las capacidades se componen, no se heredan | `DataReader` + `DataWriter` son interfaces segregadas que un mismo tipo puede implementar ambas, una, o ninguna |
| **Infraestructura Cross-Cutting** | Memoria, logging, observabilidad son capas, no módulos | Se integran via lifecycle hooks, no modificando el core |
| **Single-User First** | No agregar complejidad de multi-tenencia hasta que sea estrictamente necesaria | Sin entidad `User`, sin `user_id` en tools, sin lógica de autorización por usuario |

### 1.2 El Augmented LLM como Base

Todo sistema agentico se construye sobre el mismo bloque fundamental:

```
LLM + Tools + Memory + Retrieval
```

El LLM es el motor de razonamiento. Las tools son las manos que le permiten actuar sobre el mundo. La memoria le da continuidad entre sesiones. Retrieval le permite acceder a conocimiento que no cabe en el contexto.

Ninguno de estos componentes es opcional en un agente real. Un LLM sin tools es un chatbot. Un LLM con tools pero sin memoria es un asistente que olvida todo al cerrar sesión.

### 1.3 Workflows vs Agentes

Anthropic establece una distinción fundamental:

- **Workflow**: Sistema donde LLMs y tools se orquestan a través de paths de código predefinidos. El desarrollador controla el flujo.
- **Agente**: Sistema donde el LLM dirige dinámicamente su propio proceso, manteniendo control sobre cómo lograr la tarea.

Nuestra arquitectura es **agent-first**: el LLM decide qué tools llamar y en qué orden, dentro de los límites que definen los módulos y el system prompt. Los workflows (como el scheduler background) existen pero son la excepción, no la regla.

---

## 2. Arquitectura de Referencia: 5 Capas

```
┌──────────────────────────────────────────────────────────────────┐
│  CANAL DE ENTRADA (whatsapp/, telegram/, etc.)                   │
│  Capa superficial - traduce protocolos externos a llamadas       │
│  agent.Chat(). No contiene lógica de negocio ni de agente.       │
└──────────────────────────┬───────────────────────────────────────┘
                           │ agent.Chat(ctx, userID, message)
┌──────────────────────────▼───────────────────────────────────────┐
│  5. ORCHESTRATION LAYER (agent/aria_chat.go)                     │
│                                                                   │
│  Controla el loop agente:                                         │
│    while goal not achieved and iterations < max:                  │
│      hooks.OnMessage()          ← pre-procesamiento              │
│      session.Send()             ← LLM call                       │
│      for each toolCall:                                           │
│        executor.Execute()       ← ToolRegistry dispatch           │
│        hooks.OnToolResult()     ← post-procesamiento             │
│        session.SendToolResults()← feedback al LLM                │
│      hooks.OnResponse()         ← post-procesamiento             │
│                                                                   │
│  hooks.OnSessionEnd()           ← cleanup / extracción batch     │
└──────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│  4. AGENT EXECUTION LAYER (agent/)                               │
│                                                                   │
│  - ToolRegistry: registro centralizado de tools con dispatch     │
│  - Executor: ejecución paralela de tool calls con errgroup       │
│  - SessionManager: working memory con TTL (30 min)               │
│  - buildCorePrompt(): ensamblado del system prompt               │
│  - HookRegistry: registro y dispatch de hooks lifecycle          │
│                                                                   │
│  NO contiene lógica de negocio. NO sabe qué hacen las tools.     │
│  Solo orquesta: message → LLM → tools → LLM → response.          │
└──────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│  3. TOOL & INTEGRATION LAYER (internal/modules/)                 │
│                                                                   │
│  Capacidades del agente implementadas como módulos Gen2:         │
│    - tasks/     → DataReader + DataWriter                        │
│    - search/    → DataReader                                     │
│    - scheduler/ → DataReader + DataWriter + backgroundModule     │
│    - memory/    → DataReader + DataWriter                        │
│    - calendar/  → (futuro)                                       │
│    - email/     → (futuro)                                       │
│                                                                   │
│  Cada módulo es independiente. Cada uno tiene su propio:         │
│  domain, tools, dispatcher (read/write), y marshal.              │
└──────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│  2. MEMORY & CONTEXT LAYER (internal/infrastructure/memory/)     │
│                                                                   │
│  - Semantic Memory: facts key-value + FTS5 search                │
│  - Episodic Memory: trazas de sesiones pasadas                   │
│  - Procedural Memory: módulos Gen2 (cómo hacer cosas)            │
│  - Working Memory: SessionManager en agent/ (context window)     │
│                                                                   │
│  Capa cross-cutting: no es un módulo, es infraestructura que     │
│  tanto los hooks como los módulos pueden usar.                   │
└──────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│  1. INFRASTRUCTURE LAYER (internal/db/, agent/llm/)              │
│                                                                   │
│  - LLM API: OpenAI, Gemini, OpenRouter via interfaz LLM/Session  │
│  - SQLite: almacenamiento persistente (tasks, reminders, facts)  │
│  - Repositorios: TaskRepository, ReminderRepository, FactRepo    │
│  - Messaging: Messenger interface + WhatsApp implementation       │
└──────────────────────────────────────────────────────────────────┘
```

### 2.1 Regla de Dependencia

Las dependencias apuntan hacia adentro. Una capa puede depender de cualquier capa inferior pero nunca de una superior:

- `whatsapp/` → `agent/` → `internal/modules/` → `internal/domain/` + `internal/db/`
- `internal/modules/` → `agent/` (solo interfaces: `PortDeps`, `Messenger`, `agentllm`)
- `internal/infrastructure/` → `internal/domain/` (solo interfaces de repositorio)

### 2.2 La Prueba Ácida

Si un cambio en `internal/modules/tasks/` requiere modificar `agent/aria_chat.go`, hay un problema de acoplamiento. La comunicación debe ser siempre a través de interfaces en `agent/port.go` o via hooks.

---

## 3. Modelo de Capacidades: Módulos Gen2

### 3.1 Las Interfaces Base

```go
// Module: base interface que todo módulo debe implementar.
type Module interface {
    Name() string
    Schema() []string          // DDL statements para auto-migración
    Init(deps PortDeps) error  // Inyección de dependencias
}

// DataReader: módulos que proveen datos al agente (read-only).
type DataReader interface {
    Module
    PromptSection(ctx context.Context, userID string) string  // Contexto inyectado en system prompt
    ReadTools() []agentllm.ToolDef                              // Tools de lectura
    Read(ctx context.Context, tool string, args map[string]any) (map[string]any, error)
}

// DataWriter: módulos que mutan estado.
type DataWriter interface {
    Module
    WriteTools() []agentllm.ToolDef
    Write(ctx context.Context, tool string, args map[string]any) (map[string]any, error)
}

// backgroundModule: módulos con loop background autónomo.
type backgroundModule interface {
    Module
    Start(ctx context.Context) error
}
```

### 3.2 Principios de Diseño de Tools

Cada tool que expone un módulo debe seguir estos principios:

1. **Single Responsibility**: una tool hace exactamente una cosa. `search_and_summarize` no existe; existen `web_search` y después el LLM decide si resume.

2. **Idempotencia**: mismos argumentos producen mismos resultados. Para operaciones con side effects, usar idempotency keys.

3. **Errores Claros**: los mensajes de error deben ser lo suficientemente detallados para que el LLM pueda decidir el próximo paso.

4. **Granularidad Adecuada**: ni tan chica que requiera 10 calls para algo simple, ni tan grande que solo sirva para un caso de uso.

5. **Descripciones como Documentación**: cada tool description debe ser tan clara como si se la diéramos a un nuevo desarrollador. Incluir ejemplos y edge cases.

### 3.3 Ciclo de Vida de un Módulo

```
Registro en main.go:
  module := tasks.New()
  modules = append(modules, module)

Provisioning en Aria.New():
  1. Schema() → DDL de migración
  2. Init(deps) → inyección de dependencias
  3. type assertion → DataReader? DataWriter? → registro de tools
  4. backgroundModule? → Start(ctx) en goroutine

Ejecución (por cada tool call del LLM):
  ToolRegistry.Execute(name, args)
    → reader.Read(ctx, name, args)   // si es DataReader
    → writer.Write(ctx, name, args)  // si es DataWriter
```

### 3.4 Categorías de Capacidades

| Categoría | Ciclo de Vida | Ubicación | Ejemplos |
|---|---|---|---|
| **Módulo de Dominio** | Tool-driven (LLM invoca) | `internal/modules/<name>/` | tasks, scheduler |
| **Módulo de Infraestructura** | Tool-driven, sin estado | `internal/modules/<name>/` | search |
| **Servicio Background** | Timer-driven (goroutine propio) | `internal/services/<name>/` | reminders, memory-extractor |
| **Infraestructura Cross-Cutting** | Usada por hooks y módulos | `internal/infrastructure/<name>/` | memory store |

---

## 4. Lifecycle Hooks

### 4.1 Definición

Los hooks son el mecanismo de extensión del agente. Permiten que código externo reaccione a eventos del ciclo de vida del agente sin modificar `agent/`. Son el equivalente a middleware en HTTP, pero para el loop agente.

### 4.2 Interfaz

```go
// AgentHook es el punto de extensión del agente.
// Cualquier componente puede implementar esta interfaz para
// reaccionar a eventos del ciclo de vida.
type AgentHook interface {
    // OnMessage se llama antes de enviar el mensaje al LLM.
    // Útil para: logging, modificar el mensaje, inyectar contexto.
    OnMessage(ctx context.Context, userID string, message string) (string, error)

    // OnResponse se llama después de cada respuesta del LLM.
    // Útil para: extraer facts, logging, detectar intenciones.
    OnResponse(ctx context.Context, userID string, response string) error

    // OnToolResult se llama después de cada tool call exitosa.
    // Útil para: actualizar memoria con resultados, auditoría.
    OnToolResult(ctx context.Context, userID string, tool string, args map[string]any, result map[string]any) error

    // OnSessionEnd se llama cuando una sesión expira o se cierra.
    // Útil para: extracción batch de facts, cleanup, consolidación.
    OnSessionEnd(ctx context.Context, userID string, transcript []Message) error
}
```

### 4.3 HookRegistry

```go
type HookRegistry struct {
    mu    sync.RWMutex
    hooks []AgentHook
}

func (r *HookRegistry) Register(hook AgentHook)
func (r *HookRegistry) OnMessage(ctx, userID, message) (string, error)
func (r *HookRegistry) OnResponse(ctx, userID, response) error
func (r *HookRegistry) OnToolResult(ctx, userID, tool, args, result) error
func (r *HookRegistry) OnSessionEnd(ctx, userID, transcript) error
```

### 4.4 Casos de Uso

| Hook | Qué permite hacer |
|---|---|
| `OnMessage` | Memoria: registrar consultas del usuario. Observabilidad: log de entrada. |
| `OnResponse` | Memoria: extraer facts de respuestas. Observabilidad: log de salida, conteo de tokens. |
| `OnToolResult` | Memoria: extraer hechos de resultados. Auditoría: registrar cada tool call. |
| `OnSessionEnd` | Memoria: consolidar facts de la sesión completa. Limpieza de recursos. |

### 4.5 Lo que NO son los hooks

- No son tools. El LLM no los invoca. Ocurren automáticamente.
- No modifican el comportamiento del agente (salvo `OnMessage` que puede modificar el mensaje).
- No son para lógica de negocio. Son para capacidades cross-cutting.

---

## 5. Arquitectura de Memoria

### 5.1 Tipos de Memoria

| Tipo | Scope | Persistencia | Implementación |
|---|---|---|---|
| **Working Memory** | Sesión actual (30 min TTL) | Volátil (en memoria) | `agent/llm/session.go` — SessionManager |
| **Semantic Memory** | Cross-sesión | SQLite FTS5 | `internal/infrastructure/memory/fact_repo.go` |
| **Episodic Memory** | Cross-sesión | SQLite | `internal/infrastructure/memory/episodic.go` (futuro) |
| **Procedural Memory** | Permanente | Código + DB | Módulos Gen2 + repositorios |

### 5.2 Semantic Memory: Primer Paso Concreto

La memoria semántica se implementa como un sistema de **facts**: pares key-value con búsqueda full-text. El LLM puede leer y escribir facts explícitamente via tools, y el hook `OnSessionEnd` extrae facts implícitamente de las conversaciones.

```
Estructura de un fact:
  key:    string (namespaced, ej: "user/name", "user/timezone", "task/priority_default")
  value:  string (el valor del fact)
  ttl:    opcional (expiración)

Tools que expone:
  save_fact(key, value, ttl?)     → DataWriter
  get_fact(key)                    → DataReader
  search_facts(query)              → DataReader (FTS5)

Extracción pasiva (via OnSessionEnd):
  El hook analiza el transcript de la sesión y extrae facts
  usando el LLM o heurísticas. Los guarda automáticamente.
```

### 5.3 Por Qué Memoria NO es un Módulo Común

Memory necesita tres cosas que un módulo normal no combina:

1. **Tools** para que el LLM interactúe → esto ES un módulo (`internal/modules/memory/`)
2. **Store** persistente → infraestructura (`internal/infrastructure/memory/`)
3. **Hooks** para extracción pasiva → integración via HookRegistry

Un módulo normal (como `tasks/`) solo necesita lo #1 y un repo. Memory necesita los tres porque es una **capa**, no una capacidad.

### 5.4 Principios de Memoria

- **Retrieval antes de razonar**: los facts relevantes se inyectan en el system prompt antes de que el LLM empiece a pensar. Esto ocurre en `PromptSection()` del módulo memory.
- **Extracción después de actuar**: los facts se extraen de las respuestas del LLM via `OnResponse` y `OnSessionEnd`.
- **Menos es más**: no guardar todo. Solo hechos verificables y útiles. La memoria se degrada si está llena de ruido.
- **TTL por defecto**: los facts tienen expiración. La información desactualizada es peor que no tener información.

---

## 6. Código de Conducta del Agente

### 6.1 Al Iniciar una Sesión, SIEMPRE:

```
PASO 1: Leer AGENTS.md (este documento)
  → Revisar principios arquitectónicos y decisiones de diseño

PASO 2: Leer ROADMAP.md
  → Obtener contexto del estado del proyecto y prioridades

PASO 3: Leer memorias de Engram del proyecto
  → Llamar engram_mem_context() para recuperar sesiones anteriores
  → Buscar decisiones recientes, bugs, patrones establecidos
  → Buscar el skill-registry del proyecto

PASO 4: Cargar skills de Go
  → go-testing, go-style-core, go-naming, go-error-handling, etc.
  → Cualquier skill específica del proyecto definida en el registry

PASO 5: NO empezar a codificar sin contexto
  → Si hay dudas sobre el estado del proyecto, preguntar primero
```

### 6.2 Durante el Desarrollo:

- **Commits profesionales**: usar conventional commits (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`). Descripciones claras del qué y por qué.
- **NO agregar atribuciones AI**: nunca incluir "Co-Authored-By" ni firmas de AI.
- **NO hacer build después de cambios**: el desarrollador humano compila cuando corresponde.
- **Verificar antes de afirmar**: "dejame verificar" si hay dudas. No asumir.
- **Justificar decisiones técnicas**: siempre explicar POR QUÉ se elige una solución, con alternativas y tradeoffs.

### 6.3 Engram: Cuándo Guardar Memorias

LLAMAR `mem_save` INMEDIATAMENTE después de:

- **Decisión arquitectónica** importante
- **Bug fix** completado (qué estaba mal, por qué, cómo se corrigió)
- **Patrón** establecido (naming, estructura, convención)
- **Configuración** o setup de entorno
- **Discovery** no obvio sobre el código base
- **Decisiones de diseño** con alternativas consideradas

AL FINAL DE CADA SESIÓN, SIEMPRE:

```
LLAMAR mem_session_summary() con:
  ## Goal
  [Qué se trabajó en esta sesión]

  ## Discoveries
  [Hallazgos técnicos, gotchas, aprendizajes no obvios]

  ## Accomplished
  [Items completados con detalles clave]

  ## Next Steps
  [Lo que queda para la próxima sesión]

  ## Relevant Files
  [Archivos creados/modificados con su propósito]
```

### 6.4 Estilo de Comunicación

- Tono profesional y directo. Sin slang, sin jerga informal.
- Explicaciones con fundamento técnico: no solo QUÉ, sino POR QUÉ y CÓMO.
- Las respuestas deben incluir: la decisión, la justificación, las alternativas consideradas, y los tradeoffs.
- Usar analogías de construcción/arquitectura cuando ayuden a explicar conceptos.

---

## 7. Flujo de Trabajo por Sesión

### 7.1 Diagrama de Flujo

```
INICIO DE SESIÓN
│
├─ 1. Leer AGENTS.md ──────────────────── principios y reglas
├─ 2. Leer ROADMAP.md ─────────────────── estado y prioridades
├─ 3. Leer Engram ──────────────────────── contexto de sesiones previas
├─ 4. Cargar skills Go ────────────────── estándares de código
│
├─ ¿Feature complejo? ──SÍ──► /sdd-new   (Spec-Driven Development)
│                             ├─ explore → propose → spec → design → tasks
│                             ├─ apply   (implementación)
│                             └─ verify → archive
│
├─ ¿Cambio pequeño? ───SÍ──► Delegar a sub-agent
│                             └─ Implementar + testear
│
├─ ¿Bug fix? ──────────SÍ──► Delegar a sub-agent
│                             ├─ Reproducir / entender
│                             ├─ Fix + test
│                             └─ mem_save (bugfix)
│
└─ FIN DE SESIÓN
    └─ mem_session_summary()
```

### 7.2 SDD (Spec-Driven Development) para Features Complejos

Features que requieren cambios en múltiples archivos o nueva funcionalidad significativa siguen SDD:

```
/sdd-new <change-name>
  ├─ explore → investigar el código base y requisitos
  ├─ propose → definir alcance, enfoque, tradeoffs
  ├─ spec    → especificación detallada con escenarios
  ├─ design  → diseño técnico con decisiones arquitectónicas
  ├─ tasks   → breakdown de implementación
  ├─ apply   → implementación de tasks (en batches)
  ├─ verify  → validación contra spec + design
  └─ archive → sync + cierre
```

No todos los cambios requieren SDD. Cambios pequeños (bugs, refactors menores) se delegan directamente a un sub-agent.

### 7.3 Estructura de Directorios

```
PersonalAssistant/
├── AGENTS.md                    ← Este documento
├── ROADMAP.md                   ← Estado y prioridades del proyecto
├── agent/                       ← Core del agente (orquestación pura)
│   ├── port.go                  ← Interfaces: Module, DataReader, DataWriter
│   ├── hooks.go                 ← AgentHook interfaz + HookRegistry
│   ├── registry.go              ← ToolRegistry
│   ├── executor.go              ← Executor
│   ├── messenger.go             ← Messenger interfaz
│   ├── context.go               ← buildCorePrompt()
│   ├── aria.go                  ← Aria struct: New, options, lifecycle
│   ├── aria_chat.go             ← Chat loop + hook dispatch
│   ├── aria_provision.go        ← Module provisioning
│   ├── limits.go                ← Max iterations, cost controls
│   └── llm/                     ← Abstracción de LLM
├── cmd/assistant/
│   └── main.go                  ← Entry point, wiring
├── internal/
│   ├── domain/                  ← Tipos e interfaces de dominio
│   ├── db/                      ← SQLite + repositorios
│   ├── modules/                 ← Capacidades del agente (tool-driven)
│   │   ├── tasks/
│   │   ├── search/
│   │   ├── scheduler/
│   │   └── memory/              ← Tools de memoria
│   ├── services/                ← Procesos background autónomos
│   │   └── reminders/           ← (desde /reminders/)
│   └── infrastructure/          ← Cross-cutting
│       └── memory/              ← Store FTS5, fact CRUD
├── whatsapp/                    ← Canal de entrada WhatsApp
└── reminders/                   ← [DEPRECATED - mover a internal/services/]
```

---

## 8. Evolución y Gobernanza

### 8.1 Cuándo Agregar una Nueva Capacidad

Una nueva capacidad (Google Calendar, Email, Browser Automation) sigue este proceso:

1. **Categorizar**: ¿Es módulo de dominio, infraestructura, o servicio background?
2. **Diseñar**: ¿Qué tools expone? ¿DataReader, DataWriter, o ambos? ¿Necesita background loop?
3. **Implementar**: Como módulo Gen2 en `internal/modules/<name>/`
4. **Registrar**: En `main.go`, agregar a la slice de módulos
5. **No tocar agent/**: Si la nueva capacidad requiere modificar `aria_chat.go`, el diseño está mal

### 8.2 Cuándo NO Usar Este Modelo

- **Tareas puramente secuenciales** con pasos fijos → usar workflow simple (prompt chaining), no un agente con tools
- **Queries stateless** (una pregunta, una respuesta) → no necesitan agente, solo un LLM
- **Operaciones batch** sobre datos → usar scripts, no el agente

### 8.3 Mantenimiento del Documento

Este documento es vivo. Toda decisión arquitectónica importante debe:

1. Documentarse en AGENTS.md (si cambia un principio fundamental)
2. Guardarse en Engram como `architecture` (para contexto de sesión)
3. Reflejarse en ROADMAP.md (si afecta prioridades)

---

## Apéndice A: Referencias

- [Building Effective Agents — Anthropic](https://www.anthropic.com/engineering/building-effective-agents)
- [How We Built Our Multi-Agent Research System — Anthropic](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Writing Effective Tools for AI Agents — Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- [Agentic AI Architecture: Enterprise Guide — Epinium](https://epinium.com/en/blog/agentic-ai-architecture/)
- [5-Layer Agent Architecture — Antigravity Lab](https://antigravitylab.net/en/articles/agents/ai-agent-system-design-complete-guide)
- [Choose a Design Pattern for Agentic AI — Google Cloud](https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)

---

## Apéndice B: Changelog Arquitectónico

| Fecha | Cambio | Decisión |
|---|---|---|
| 2026-05-13 | Creación de AGENTS.md | Documentar principios arquitectónicos, hook system, modelo de memoria, y código de conducta del agente |
