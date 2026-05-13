# Skill Registry — PersonalAssistant

## Project-Level Conventions
- **AGENTS.md**: Not found
- **CLAUDE.md**: Not found
- **GEMINI.md**: Not found
- **.agent/**: Not found

## User-Level Skills

### Go Skills (from ~/.agents/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| go-code-review | ~/.agents/skills/go-code-review/SKILL.md | Review Go code against community style standards |
| go-concurrency | ~/.agents/skills/go-concurrency/SKILL.md | Writing concurrent Go code — goroutines, channels, mutexes |
| go-context | ~/.agents/skills/go-context/SKILL.md | Working with context.Context in Go |
| go-control-flow | ~/.agents/skills/go-control-flow/SKILL.md | Conditionals, loops, switch statements in Go |
| go-data-structures | ~/.agents/skills/go-data-structures/SKILL.md | Slices, maps, arrays in Go |
| go-declarations | ~/.agents/skills/go-declarations/SKILL.md | Variable, constant, struct, map declarations |
| go-defensive | ~/.agents/skills/go-defensive/SKILL.md | Hardening Go code at API boundaries |
| go-documentation | ~/.agents/skills/go-documentation/SKILL.md | Writing docs for Go packages, types, functions |
| go-error-handling | ~/.agents/skills/go-error-handling/SKILL.md | Error return, wrap, handle patterns |
| go-functional-options | ~/.agents/skills/go-functional-options/SKILL.md | Constructor/config functional options pattern |
| go-functions | ~/.agents/skills/go-functions/SKILL.md | Function organization and signatures |
| go-generics | ~/.agents/skills/go-generics/SKILL.md | Generic functions and types |
| go-interfaces | ~/.agents/skills/go-interfaces/SKILL.md | Defining and implementing interfaces |
| go-linting | ~/.agents/skills/go-linting/SKILL.md | Setting up golangci-lint, Go checks |
| go-logging | ~/.agents/skills/go-logging/SKILL.md | Structured logging with slog |
| go-naming | ~/.agents/skills/go-naming/SKILL.md | Idiomatic naming for Go identifiers |
| go-packages | ~/.agents/skills/go-packages/SKILL.md | Package organization and structure |
| go-performance | ~/.agents/skills/go-performance/SKILL.md | Optimization and benchmarking |
| go-style-core | ~/.agents/skills/go-style-core/SKILL.md | Formatting, nesting, core style principles |
| go-testing | ~/.agents/skills/go-testing/SKILL.md | Writing Go tests — table-driven, subtests, helpers |

### Go Testing (Gentleman.Dots variant, from ~/.config/opencode/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| go-testing | ~/.config/opencode/skills/go-testing/SKILL.md | Go testing patterns including Bubbletea TUI testing |

### TUI/CLI Skills (from ~/.agents/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| building-glamorous-tuis | ~/.agents/skills/building-glamorous-tuis/SKILL.md | Build terminal UIs with Charmbracelet |
| charm-stack | ~/.agents/skills/charm-stack/SKILL.md | Bubbletea, Bubbles, Lipgloss, Huh |
| gentleman-bubbletea | ~/.agents/skills/gentleman-bubbletea/SKILL.md | Bubbletea TUI patterns for Gentleman.Dots |

### Notion Skills (from ~/.agents/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| notion-api | ~/.agents/skills/notion-api/SKILL.md | Notion API REST calls |
| notion-spec-to-implementation | ~/.agents/skills/notion-spec-to-implementation/SKILL.md | Turn Notion specs into implementation plans |

### Other Skills (from ~/.agents/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| deploy-to-vercel | ~/.agents/skills/deploy-to-vercel/SKILL.md | Deploy applications to Vercel |
| find-skills | ~/.agents/skills/find-skills/SKILL.md | Discover and install agent skills |

### SDD Skills (from ~/.config/opencode/skills/)
| Skill | Path | Description |
|-------|------|-------------|
| sdd-init | ~/.config/opencode/skills/sdd-init/SKILL.md | Initialize SDD context in project |
| sdd-explore | ~/.config/opencode/skills/sdd-explore/SKILL.md | Explore ideas before committing to change |
| sdd-propose | ~/.config/opencode/skills/sdd-propose/SKILL.md | Create change proposal |
| sdd-spec | ~/.config/opencode/skills/sdd-spec/SKILL.md | Write specifications with requirements |
| sdd-design | ~/.config/opencode/skills/sdd-design/SKILL.md | Create technical design documents |
| sdd-tasks | ~/.config/opencode/skills/sdd-tasks/SKILL.md | Break down change into implementation tasks |
| sdd-apply | ~/.config/opencode/skills/sdd-apply/SKILL.md | Implement tasks from the change |
| sdd-verify | ~/.config/opencode/skills/sdd-verify/SKILL.md | Validate implementation matches specs |
| sdd-archive | ~/.config/opencode/skills/sdd-archive/SKILL.md | Archive completed change |

### Context Engineering Skills (from ~/.claude/skills/context-engineering-collection/)
| Skill | Path | Description |
|-------|------|-------------|
| context-fundamentals | ~/.claude/skills/context-engineering-collection/skills/context-fundamentals/SKILL.md | Understanding context windows and agent architecture |
| context-optimization | ~/.claude/skills/context-engineering-collection/skills/context-optimization/SKILL.md | Reduce token costs, improve context efficiency |
| context-compression | ~/.claude/skills/context-engineering-collection/skills/context-compression/SKILL.md | Summarization, compaction, token reduction |
| context-degradation | ~/.claude/skills/context-engineering-collection/skills/context-degradation/SKILL.md | Diagnose context problems, lost-in-middle |
| filesystem-context | ~/.claude/skills/context-engineering-collection/skills/filesystem-context/SKILL.md | File-based context management |
| multi-agent-patterns | ~/.claude/skills/context-engineering-collection/skills/multi-agent-patterns/SKILL.md | Design multi-agent systems, supervisor pattern |
| hosted-agents | ~/.claude/skills/context-engineering-collection/skills/hosted-agents/SKILL.md | Background agents, sandboxed execution |
| latent-briefing | ~/.claude/skills/context-engineering-collection/skills/latent-briefing/SKILL.md | Cross-agent memory, KV cache compaction |
| memory-systems | ~/.claude/skills/context-engineering-collection/skills/memory-systems/SKILL.md | Agent memory frameworks and persistence |
| evaluation | ~/.claude/skills/context-engineering-collection/skills/evaluation/SKILL.md | Evaluate agent performance, quality gates |
| advanced-evaluation | ~/.claude/skills/context-engineering-collection/skills/advanced-evaluation/SKILL.md | LLM-as-judge, pairwise comparison |
| bdi-mental-states | ~/.claude/skills/context-engineering-collection/skills/bdi-mental-states/SKILL.md | BDI architecture, cognitive agents |
| tool-design | ~/.claude/skills/context-engineering-collection/skills/tool-design/SKILL.md | Design agent tools, MCP tools |
| project-development | ~/.claude/skills/context-engineering-collection/skills/project-development/SKILL.md | Start LLM projects, pipeline architecture |

### Context Engineering Examples (from ~/.claude/skills/context-engineering-collection/)
| Skill | Path | Description |
|-------|------|-------------|
| interleaved-thinking | ~/.claude/skills/context-engineering-collection/examples/interleaved-thinking/SKILL.md | Debug/optimize AI agents via reasoning traces |
| digital-brain | ~/.claude/skills/context-engineering-collection/examples/digital-brain-skill/SKILL.md | Personal brand, content creation, voice consistency |
| book-sft-pipeline | ~/.claude/skills/context-engineering-collection/examples/book-sft-pipeline/SKILL.md | Fine-tune on books, SFT dataset creation |
| comprehensive-research-agent | ~/.claude/skills/context-engineering-collection/examples/interleaved-thinking/generated_skills/comprehensive-research-agent/SKILL.md | Thorough validation in research tasks |
