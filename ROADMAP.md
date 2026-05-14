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

- [x] **`web_search(query)`** — Búsqueda Google Search Grounding vía Gemini REST API. Reemplazó DuckDuckGo.
- [x] **`fetch_url(url)`** — HTTP GET + Content-Type check + html→texto + truncado a 8K chars
- [x] **Modo dual Gemini** — Cuando `GEMINI_API_KEY` está configurada:
  - Chat: function declarations (tasks, scheduler) + `web_search` tool via Gemini REST
  - Scheduler: GoogleSearch built-in para prompts de texto plano automáticos
- [x] **System prompt actualizado** — Instrucciones de búsqueda simplificadas

> ⚠️ **Limitación conocida**: Gemini 2.5 Flash NO permite GoogleSearch + function declarations. El split por tipo de sesión lo resuelve.
> 
> **Próximo paso**: Al migrar a Gemini 3+, combinar ambas en una sesión.

**Archivos:** `agent/llm/gemini.go`, `agent/llm/gemini_session.go`, `internal/modules/search/gemini_search.go`
**Dependencia nueva:** `google.golang.org/genai`

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

### 3. ✅ Memoria Persistente entre Sesiones

Que el agente recuerde conversaciones pasadas, preferencias del usuario, y contexto entre sesiones.

- [x] **SQLite FTS5 para búsqueda semántica** — Tabla `memory_facts` + `memory_fts` virtual con triggers de sync automáticos
- [x] **Tool `save_fact(key, value, ttl?)` / `get_fact(key)` / `search_memory(query)`** — CRUD completo de facts vía tools del LLM
- [x] **Inyección automática en system prompt** — `PromptSection()` inyecta todos los facts activos directamente en el prompt al iniciar sesión. El LLM los recibe sin tener que llamar herramientas.
- [ ] **Extracción automática de facts** — Hook `OnSessionEnd` analiza el transcript y extrae facts automáticamente. Pendiente de implementar.

> ⚠️ **Fix aplicado (14/05/2026)**: Originalmente `PromptSection()` solo describía las tools, no inyectaba los valores. Corrección: ahora llama `GetAllFacts()` y los incluye como contexto directo con instrucción "DEBES usarla en tus respuestas".

**Archivos:** `internal/modules/memory/{module,read,write,tools}.go`, `internal/infrastructure/memory/store.go`

### 4. ✅ Deudas (Debts) — Integrado desde fito/

Módulo completo de gestión de deudas, portado desde el proyecto `fito/` y simplificado para single-user (sin UserID, nombre del deudor como identificador).

- [x] **CRUD completo de deudas** — 6 tools: `create_debt`, `get_all_debts`, `get_debt_by_id`, `update_debt`, `update_debt_state`, `delete_debt`
- [x] **CRUD completo de pagos** — 5 tools: `add_debt_payment`, `get_debt_payments`, `update_debt_payment`, `delete_debt_payment`
- [x] **Auto-recalculo de estado** — Al agregar/eliminar pagos, el estado se recalcula automáticamente (PENDING → PARTIAL → PAID)
- [x] **Validaciones** — Montos negativos, sobrepagos, estados inválidos
- [x] **Precisión monetaria** — `shopspring/decimal` en dominio, `int64` cents en DB, formateo a string para el LLM
- [x] **PromptSection con deudas pendientes** — LEFT JOIN con pagos para mostrar total adeudado vs pagado

> **Decisión arquitectónica**: Single-user. No existe tabla `users`. El deudor se identifica por su nombre en el campo `Name`. La tool `get_debts_by_user` se eliminó por ser redundante.

**Archivos:** `internal/modules/debts/{module,tools,read,write,marshal}.go`, `internal/db/debt_repo.go`, `internal/db/debt_payment_repo.go`
**Dependencia nueva:** `github.com/shopspring/decimal`

### 5. Proactividad

Que el asistente hable sin que le pregunten.

- [ ] **Engine de proactividad** — Cada N minutos revisa tareas próximas a vencer, recordatorios pendientes, etc.
- [ ] **Saludo diario** — "Buenos días, tenés 3 tareas pendientes y 1 recordatorio para hoy"
- [ ] **Alertas de vencimiento** — "⚠️ La tarea X vence mañana"
- [ ] **WhatsApp proactive send** — El bot puede iniciar conversación (no solo responder)

**Archivos:** `internal/proactive/`, integración con `reminders/service.go`

### 6. Calendario (Google Calendar)

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

---

## 🛠️ Infraestructura y Operaciones

### Deploy en Termux (Android)

El asistente corre en un teléfono Android vía Termux, accesible por SSH (puerto 8022) a través de Tailscale.

| Componente | Detalle |
|---|---|
| **Dispositivo** | Samsung Galaxy A36 5G — `galaxy-a36-5g` (Tailscale: `100.68.21.101`) |
| **SSH** | Puerto `8022` (Termux no puede usar <1024 sin root) |
| **Runtime** | `tmux` session + `bash ~/start-assistant.sh` |
| **Logs** | `~/pa.log` (stdout + stderr via `tee`) |
| **Binario** | `~/assistant` (cross-compiled `linux/arm64`) |
| **DB** | `~/PersonalAssistant/assistant.db` (SQLite WAL) |

### Startup Script (`~/start-assistant.sh`)

```bash
#!/data/data/com.termux/files/usr/bin/bash
cd ~/PersonalAssistant
set -a
source .env
set +a
export GODEBUG=netdns=go
exec ~/assistant 2>&1 | tee ~/pa.log
```

### DNS override (aplicado en código)

**Problema**: Tailscale reemplaza el DNS del sistema por su proxy local (`127.0.0.1:53`). Cuando el dispositivo Android entra en deep sleep, Tailscale se suspende y el proxy DNS muere. Esto impide que whatsmeow reconecte el WebSocket de WhatsApp.

**Solución**: Override global de `net.DefaultResolver` en `cmd/assistant/main.go`:

```go
var dnsServers = []string{"8.8.8.8:53", "1.1.1.1:53"}

func init() {
    net.DefaultResolver = &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
            d := net.Dialer{Timeout: 5 * time.Second}
            for _, srv := range dnsServers {
                conn, err := d.DialContext(ctx, "udp", srv)
                if err == nil {
                    return conn, nil
                }
            }
            return nil, fmt.Errorf("dns: all servers unreachable (%v)", dnsServers)
        },
    }
}
```

Esto hace que TODA resolución DNS de la aplicación vaya directo a `8.8.8.8:53` (con fallback a `1.1.1.1:53`) sin pasar por el resolver del sistema ni por Tailscale.

> ⚠️ **Limitación conocida**: Este override no puede resolver dominios de la tailnet (MagicDNS `*.ts.net`) ni dominios `.local`. Es intencional — PersonalAssistant solo necesita dominios públicos. Si en el futuro se requiere, habrá que agregar lógica condicional.

### Arquitectura de Módulos (Gen2)

Cada capacidad del agente es un módulo autocontenido que implementa las interfaces del `agent`:

```
Module        → Name(), Schema(), Init(PortDeps)
DataReader    → Module + PromptSection() + ReadTools() + Read()
DataWriter    → Module + WriteTools() + Write()
```

| Módulo | Tools | Reader | Writer | Background |
|---|---|---|---|---|
| `tasks` | 5 | ✅ | ✅ | ❌ |
| `search` | 2 | ✅ | ❌ | ❌ |
| `scheduler` | 6 | ✅ | ✅ | ✅ (ticker 60s) |
| `memory` | 3 | ✅ | ✅ | ❌ |
| `debts` | 11 | ✅ | ✅ | ❌ |

---

## 🤝 Cómo Contribuir

1. Fork del repo: `github.com/carlospereira5/PersonalAssistant`
2. Crear branch: `feat/nombre-del-feature`
3. Commit convencional: `feat: descripción`
4. PR con descripción clara de qué cambia y por qué
