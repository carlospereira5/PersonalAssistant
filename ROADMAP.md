# Personal Assistant — Roadmap

> Asistente personal ligero en Go. Tasks + reminders vía WhatsApp + LLM agent loop.

## 🚀 Fase 1 — Actual (Completado)

- [x] **Task CRUD** — Módulo Gen2 del agente: `create_task`, `get_all_tasks`, `update_task_status`, `delete_task`
- [x] **Recordatorios vía WhatsApp** — Background service cada 30s que envía reminders al admin JID detectado automáticamente
- [x] **Auto-descubrimiento de admin JID** — El bot guarda el JID del administrador cuando recibe el primer mensaje (sin config manual)
- [x] **WhatsApp integration** — whatsmeow con QR login, modo DM/grupo, transcripción de audio (Whisper)
- [x] **LLM tool calling** — Aria orchestrator con function calling, streaming, sesiones multi-turno (30min TTL)
- [x] **Multi-provider LLM** — OpenAI, OpenRouter, Google Gemini, Groq vía base URL configurable
- [x] **Infisical secrets** — Gestión centralizada de API keys y config
- [x] **Arquitectura single-user** — Sin entidad User, sin user_id en tools, todo simplificado para un único administrador
- [x] **GitHub público** — `github.com/carlospereira5/PersonalAssistant` con MIT license

---

## 🎯 Fase 2 — Prioridad Alta (P0)

### 1. Web Search

El feature más pedido en TODOS los asistentes personales open source.

- [x] **Módulo Gen2 `search`** — `internal/modules/search/` con `web_search(query)` y `fetch_url(url)`
- [x] **Tool `web_search(query)`** — Búsqueda DuckDuckGo via `github.com/evgensoft/ddgo`, retorna hasta 5 resultados
- [x] **Tool `fetch_url(url)`** — HTTP GET + validación Content-Type + conversión HTML→texto (html2text) + truncado a 8K chars
- [x] **Integración en system prompt** — Herramientas listadas con instrucciones de uso en cadena
- [x] **`get_current_time()`** — Tool añadida al módulo tasks para que el LLM resuelva fechas sin depender de su conocimiento interno

> ⚠️ **Limitación conocida**: `fetch_url` NO ejecuta JavaScript. Sitios SPA (React, Vue) que cargan contenido dinámicamente devuelven solo la navegación vacía. Funciona bien con contenido estático: Wikipedia, documentación, blogs, noticias.
> 
> **Próximos pasos**: Explorar integración con wttr.in para clima, o agregar un provider específico para sitios SPA vía chromedp/playwright si es necesario.

**Archivos:** `internal/modules/search/{module,tools,read,search,fetch}.go`
**Dependencias:** `github.com/evgensoft/ddgo`, `github.com/k3a/html2text`

### 2. Scheduled Routines (Cron)

Rutinas programadas que ejecutan acciones automáticamente usando expresiones cron. El LLM traduce lenguaje natural a cron expressions.

- [x] **Scheduler daemon** — Background service (backgroundModule) que revisa rutinas cada 60s
- [x] **Tabla `scheduled_routines`** — Con cron expression, prompt, estado (active/paused), last_run_at, last_error, CHECK constraint
- [x] **Tool `create_routine(cron, prompt)`** — Con validación de cron expression via robfig/cron/v3
- [x] **Tool `list_routines()` / `delete_routine(id)` / `pause_routine(id)` / `resume_routine(id)`** — CRUD completo
- [x] **Ejecución vía LLM** — Cada rutina ejecuta su prompt contra el LLM y entrega el resultado vía Messenger
- [x] **Auto-recuperación** — Si una rutina falla, loguea el error, guarda last_error y continúa
- [ ] **Morning briefing** — Rutina pre-definida que lista tareas del día (próximo feature)

> ⚠️ **LLM traduce lenguaje natural a cron**: el usuario escribe "cada día a las 8", el LLM llama `create_routine(cron: "0 8 * * *", prompt: "...")`. No hay parser de lenguaje natural — el modelo se encarga.
>
> **Limitación conocida**: Las rutinas ejecutan prompts de texto al LLM, no herramientas arbitrarias. Para ejecutar tools específicas, extender el módulo.

**Archivos:** `internal/modules/scheduler/{module,tools,read,write,exec,marshal}.go`, `internal/db/scheduler_repo.go`
**Dependencias:** `github.com/robfig/cron/v3`

---

## 🔮 Fase 3 — Prioridad Media (P1)

### 3. Memoria Persistente entre Sesiones

Que el agente recuerde conversaciones pasadas, preferencias del usuario, y contexto entre sesiones.

- [ ] **Extracción automática de facts** — El agente extrae hechos relevantes de cada conversación (preferencias, datos personales, decisiones)
- [ ] **SQLite FTS5 para búsqueda semántica** — Búsqueda full-text sobre historial de conversaciones
- [ ] **Tool `save_fact(key, value)` / `get_fact(key)`** — Memoria explícita manipulable por el LLM
- [ ] **Inyección automática en system prompt** — Facts relevantes se inyectan en el prompt según el contexto de la conversación

**Archivos:** `internal/memory/`, `internal/db/memory_repo.go`

### 4. Proactividad

Que el asistente hable sin que le pregunten.

- [ ] **Engine de proactividad** — Cada N minutos revisa tareas próximas a vencer, recordatorios pendientes, etc.
- [ ] **Saludo diario** — "Buenos días, tenés 3 tareas pendientes y 1 recordatorio para hoy"
- [ ] **Alertas de vencimiento** — "⚠️ La tarea X vence mañana"
- [ ] **WhatsApp proactive send** — El bot puede iniciar conversación (no solo responder)

**Archivos:** `internal/proactive/`, integración con `reminders/service.go`

### 5. Calendario (Google Calendar)

- [ ] **OAuth2 con Google Calendar** — Leer eventos, crear eventos
- [ ] **Tool `calendar_list(date)`** — "Qué tengo mañana"
- [ ] **Tool `calendar_create(summary, date, duration)`** — "Agendá reunión para el viernes a las 15"
- [ ] **Tool `calendar_search(query)`** — Buscar eventos por texto

**Dependencias:** `golang.org/x/oauth2`, `googleapis/google-api-go-client/calendar/v3`

---

## 🧱 Fase 4 — Prioridad Baja (P2)

### 6. Skills System

Que el agente pueda extender sus capacidades en caliente.

- [ ] **SKILL.md format** — Skills definidas en markdown con frontmatter (name, description, tools)
- [ ] **Hot-reload** — Detectar nuevos archivos de skills en un directorio y cargarlos sin reiniciar
- [ ] **Tool `install_skill(url)`** — Instalar skills desde GitHub
- [ ] **Tool `list_skills()` / `remove_skill(name)`** — Gestión de skills

### 7. Browser Automation

- [ ] **Playwright/chromedp integration** — Navegador headless para automatización web
- [ ] **Tool `browser_navigate(url)`** — Ir a una URL
- [ ] **Tool `browser_screenshot()`** — Capturar pantalla
- [ ] **Tool `browser_click(selector)` / `browser_type(text)`** — Interactuar con elementos

### 8. Email (Gmail)

- [ ] **OAuth2 con Gmail** — Leer y enviar correos
- [ ] **Tool `email_list(limit)`** — Últimos N correos
- [ ] **Tool `email_search(query)`** — Buscar correos
- [ ] **Tool `email_send(to, subject, body)`** — Enviar correo

### 9. Notas / Journaling

- [ ] **Tool `save_note(title, content)`** — Guardar notas rápidas
- [ ] **Tool `list_notes()` / `search_notes(query)`** — Buscar notas
- [ ] **Tool `daily_log()`** — Diario personal: "qué hice hoy"

---

## 🌌 Fase 5 — Visión (P3)

### 10. Mejoras en Experiencia

- [ ] **Voice TTS** — Respuestas de voz usando ElevenLabs o OpenAI TTS
- [ ] **Image generation** — DALL-E / Stable Diffusion para crear imágenes bajo demanda
- [ ] **Multi-canal** — Telegram, Discord además de WhatsApp
- [ ] **Desktop app** — Tauri wrapper para el asistente

### 11. Sistema Multi-agente

- [ ] **Delegación entre agentes** — Agent teams con supervisor routing
- [ ] **Agentes especializados** — Investigador, escritor, analista
- [ ] **Task board** — Kanban-style task management vía el propio agente

### 12. Observabilidad

- [ ] **Token usage tracking** — Cuántos tokens gasta cada conversación/rutina
- [ ] **Tool call audit** — Historial de herramientas invocadas
- [ ] **Cost estimation** — Presupuesto mensual de API

---

## 📐 Principios de Arquitectura

| Principio | Por qué |
|-----------|---------|
| **Single-user first** | Sin complejidad innecesaria de multi-tenencia |
| **Módulos Gen2** | DataReader + DataWriter siguiendo convenciones del agent |
| **Infisical para secrets** | No hay API keys en el repo ni en env files |
| **SQLite, sin servidor** | Cero infraestructura, single binary |
| **whatsmeow, no Business API** | Sin costos ni aprobaciones de Meta |
| **Un sóllo binario** | `go build` produce un único binario estático |

---

## 🤝 Cómo Contribuir

1. Fork del repo: `github.com/carlospereira5/PersonalAssistant`
2. Crear branch: `feat/nombre-del-feature`
3. Commit convencional: `feat: descripción`
4. PR con descripción clara de qué cambia y por qué
