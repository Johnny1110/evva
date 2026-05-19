# evva — Roadmap Additions & Project Health Check

_A consolidated report from a survey of `ref/src/` (Claude Code TypeScript source) and an audit of `internal/` (evva Go source)._

This document has two parts:

1. **Part 1 — Roadmap feature additions.** Features and design ideas from `ref/` that aren't yet covered by the existing phases in `CLAUDE.md`. Split into _must-have_ (real gaps a v1 coding agent will hit) and _nice-to-have_ (defer).
2. **Part 2 — Project health check.** Architectural findings from auditing `internal/`. Confused boundaries, closed-enum smells, dead code, missing tests — anything that wants refactoring before later phases land on top.

The roadmap source of truth is `CLAUDE.md`. Anything already on the roadmap is excluded from Part 1 — what's listed here are _additions_.

---

## Part 1 — Roadmap feature additions

### Must-have additions

These are real gaps. Without them the existing phases will either degrade in long sessions, reinvent infrastructure, or ship with quiet correctness holes.

| # | Feature | Why it matters | Ref path | Size |
|---|---|---|---|---|
| M1 | **Auto-compaction service** | Any session longer than ~30 turns blows the context window today. evva has no compaction; ref has reactive + auto compaction, summary rewriting, cache-aware message normalization. Without this, profile switching (Phase 6), plan mode (Phase 7), and the user-profile agent (Phase 9) all degrade badly. | `services/compact/` | Large |
| M2 | **LSP tool** (symbol lookup, hover, diagnostics) | The single biggest "coding agent" leg-up. evva is otherwise mostly Read+Grep+Edit, which the model has to chain manually. LSP turns "find references to `Foo.Bar`" into one call. | `tools/LSPTool/`, `services/lsp/` | Medium–Large |
| M3 | **Brief tool** | One short tool that lets the model snapshot its own context summary mid-session. Trivial to implement; massively helps long agentic runs and is the manual escape hatch when auto-compaction misfires. | `tools/BriefTool/` | Small |
| M4 | **Layered settings** (global → project → local, with merge) | evva has `configs/` but no documented precedence. Phase 3 (permissions), Phase 4 (hooks), Phase 6 (profiles) all want per-project overrides; without a settings layer they will each reinvent it. | `utils/settings/` | Medium |
| M5 | **Token estimation + cost tracker** | Phase 12 added model effort but no token counting or per-provider $$ accounting. Users running on paid providers need this in the status bar. | `cost-tracker.ts`, `services/tokenEstimation.ts` | Medium |
| M6 | **Output styles** (markdown-driven prompt templates) | A persona has one system prompt; an output style is "the same persona writes terser / more formal / pirate-mode." This is the right separation. Costs almost nothing (just files + a sysprompt include). Fits cleanly next to Phase 6. | `outputStyles/` | Small |
| M7 | **Migrations framework** | Once Phase 6 ships persona configs and Phase 3 ships permission files, evva will need to migrate them across versions. Cheap to add now, painful to retrofit later. | `migrations/` | Small |
| M8 | **Notebook Edit tool** | Phase 1 ported Read of `.ipynb`; Edit is missing. Quick win that closes the notebook story. | `tools/NotebookEditTool/` | Small–Medium |
| M9 | **Forked-agent utility** with prompt-cache sharing | Phase 2 ships AgentTool, but ref has a real `utils/forkedAgent.ts` that shares the prompt cache with the parent. Big latency/cost win for spawned subagents. | `utils/forkedAgent.ts` | Medium |

### Nice-to-have additions

These are real features, but they don't pay for themselves until evva v1 is solid. Defer to v2+.

| # | Feature | Why deferrable | Ref path |
|---|---|---|---|
| N1 | **Magic Docs** (auto-refreshing project markdown via header marker) | Cool but niche; EVVA.md already covers project memory. | `services/MagicDocs/` |
| N2 | **Prompt suggestion / speculation** (predict next user prompt) | UX polish, costs forked-agent calls. | `services/PromptSuggestion/` |
| N3 | **Tips scheduler** (contextual help hints) | Pure UX. | `services/tips/` |
| N4 | **Keybinding file system** | Bubbletea v2 already has hard-coded keys that work fine. Worth doing once there's user demand. | `keybindings/` |
| N5 | **Plugin system** | Phase 13 already covers MCP + skills; a full plugin marketplace is a layer above. | `plugins/` |
| N6 | **Auto-Dream / Session Memory** (background memory consolidation) | Adjacent to Phase 9 (user-profile agent) — fold into that phase if extending it, otherwise defer. | `services/autoDream/`, `services/SessionMemory/` |
| N7 | **Coordinator mode** (async sub-agents w/ synthetic tools) | Multi-agent orchestration. Higher leverage once personas mature. | `coordinator/coordinatorMode.ts` |
| N8 | **REPL tool** (interactive Node/Python shells) | Bash + run_in_background mostly covers this. | `tools/REPLTool/` |
| N9 | **PowerShell tool** | Only matters on Windows. | `tools/PowerShellTool/` |
| N10 | **Schedule cron tool** | Niche; users with cron needs can shell out. | `tools/ScheduleCronTool/` |
| N11 | **Config tool** (model-callable settings reads) | Small. Pairs with M4 layered settings. | `tools/ConfigTool/` |
| N12 | **Diagnostic tracking / analytics queue** | Telemetry isn't valuable until evva has users. | `services/analytics/`, `services/diagnosticTracking.ts` |
| N13 | **Policy limits service** | Org-level feature restrictions — for enterprise deploys. | `services/policyLimits/` |
| N14 | **Voice mode** (STT + TTS) | Whole separate UX axis; ship after the core is solid. | `voice/`, `services/voice*.ts` |
| N15 | **VCR / replay test infrastructure** | Worth adding once test coverage gaps (see Part 2) are filled. | `services/vcr.ts` |

### Explicitly skipped (already out of scope per CLAUDE.md)

For completeness, the following ref features were considered and rejected:

- **OAuth / Team memory sync / Settings sync** — multi-device sync is out of scope for v1.
- **Remote session manager / SDK message adapter** — same reason; "Teams / SendMessage" is already documented as v3+.
- **Buddy / mascot** — pure decoration.
- **Growthbook feature flags** — no production deployment yet to gate.
- **Native-TS bridge, upstream proxy** — TS-specific runtime concerns that don't apply to a Go port.

### Suggested phase placement

This is a sketch, not a decision:

- **M1 (auto-compaction)** is large enough to warrant its own phase — slot it _before_ Phase 6 (profile switch), since switching personas across a long session needs compaction to be meaningful.
- **M2 (LSP)** deserves its own phase — meaningful integration work, not a small port.
- **M4 (settings) + M7 (migrations)** pair naturally with Phase 3 (permissions).
- **M5 (cost tracker)** is a small dedicated chunk — could fold into a Phase 12 follow-up or sit on its own.
- **M3 (Brief), M6 (output styles), M8 (notebook edit), M9 (forked-agent)** are small enough to fold into existing phases:
- Brief → near Phase 5 (TodoWrite) or its own micro-phase.
- Output styles → Phase 6 (profile manager).
- Notebook edit → addendum to Phase 1.
- Forked-agent → Phase 2 (AgentTool polish).

---

## Part 2 — Project health check

The audit covered: package boundary leaks, duplicated logic, closed-enum smells, misplaced responsibility, tool-family hygiene, sysprompt structure, LLM client interface tightness, UI v1 vs v2, profile/loader state, session model, test coverage, logging, and naming churn from incomplete phases.

Findings group into four tiers. Three observations are explicitly _healthy_ — listed at the end.

### Critical — blocks upcoming phases

| Finding | Where | Impact | Refactor size |
|---|---|---|---|
| **Phase 5 not started: task tools still live** | `internal/tools/task/`, `internal/agent/profiles.go:131`, `internal/agent/sysprompt/main_agent.go:34,90,122–138`, `internal/agent/sysprompt/toolnames.go:37–39` | Deferred-tools list and main prompt still advertise six `task_*` tools; TodoWrite cannot land on top of this without a name clash. | Small |
| **No AgentRegistry — profiles are global function constructors** | `internal/agent/profiles.go` (Main, Explore, General) | Phase 6 (`/profile` picker, user-authored personas, `evva → nono` delegation) is blocked. There's nothing to enumerate. | Medium |
| **Hardcoded `subagent_type` switch** | `internal/agent/spawn.go:140–157` | Only `explore` / `general-purpose` / `general` are spawnable; user agents from `<EVVA_HOME>/agents/` can't be spawned even if Phase 6 ships the loader. Same phase, same fix as the registry. | Medium (pairs with profiles registry) |
    | **`toolset.buildOne` is a 100-LOC closed-enum switch** | `internal/toolset/toolset.go:273–373` | Phase 2 promised `Registry.Register(name, factory)`. External tool packs can't register without forking. Also blocks Phase 13 MCP-registered tools. | Large |

### High — maintenance risk

| Finding | Where | Impact | Refactor size |
|---|---|---|---|
| **`Tool.Execute` has no logger** | `internal/tools/tool.go:16` + every tool impl | Phase 1c gap. Tool debug/error info goes nowhere. Bash hangs and Edit errors are invisible in prod logs. | Medium (touches all ~44 tool impls) |
| **V1 bubbletea UI is dead code** | `internal/ui/bubbletea/` (~4.3K LOC); `cmd/evva/main.go` only imports v2 | Confuses contributors, doubles maintenance surface (see diff dup below). | Small (delete after grep-confirming no imports) |
| **Multimodal parity uneven across providers** | `internal/llm/claude/client.go` (full), `deepseek/client.go:495` (text fallback), `openai/client.go` (none), `ollama/client.go` (minimal) | Phase 1b's promise of round-tripping image `tool_result`s only fully holds for Anthropic. Image reads silently degrade on other providers. | Medium |
| **Big files with no tests** | `internal/ui/bubbletea/app.go` (1,048 LOC), `internal/ui/bubbletea/config_panel.go` (392), `internal/agent/state_machine.go` (333) | `state_machine.go` has no coverage — any change to the agent loop is high-risk. The UI files are partially mitigated by being v1 (delete). | Medium |

### Medium — duplication / cleanup

| Finding | Where | Impact | Refactor size |
|---|---|---|---|
| **Diff rendering duplicated v1/v2** | `internal/ui/bubbletea/diff.go` (84 LOC) vs `internal/ui/bubbletea_v2/components/diff/diff.go` (153 LOC) | Auto-resolved once v1 is deleted. Until then, two places to fix bugs. | Small (folds into v1 delete) |
| **`tools/util/` is a junk drawer** | `internal/tools/util/` (Calc + JSONQuery, ~897 LOC together) | Two unrelated tools sharing one package. Either split (`tools/math/`, `tools/json/`) or rename `util/` to something descriptive. | Small |
| **`tools/monitor/` is a stub** | `internal/tools/monitor/` | Either delete or build out. Currently neither. | Small |

### Healthy — confirmed clean

These were checked and are good as-is:

- `internal/observable/` has zero external deps (only `sync`, `time`). Clean.
- `internal/agent/` does not import `internal/ui/`. Boundary holds.
- No tool package imports `internal/llm`. Boundary holds.
- `internal/agent/sysprompt/toolnames_link_test.go` already catches prompt-vs-tool-name drift. Keep extending it as new tool names are added.

### Suggested cleanup order

Cheapest sequence that unblocks the most upcoming phases:

1. **Delete bubbletea v1** — small, removes 4.3K LOC + diff duplication.
2. **Finish Phase 5 task cleanup** — small, unblocks TodoWrite.
3. **Add logger to `Tool.Execute`** — medium, mechanical, unlocks debugging for everything after.
4. **Build `AgentRegistry` + replace `spawn.go` switch** — medium, prereq for Phase 6.
5. **Replace `toolset.buildOne` switch with `Registry.Register`** — large, Phase 2 deliverable. Do this before Phase 13 ships MCP-registered tools.
6. **Multimodal provider parity tests + documented fallback** — medium, prevents silent regressions.
7. **Backfill `state_machine.go` tests** — medium, before any further agent loop changes.

---

## Appendix — Method

- **Ref survey**: directory listing of every subtree under `ref/src/`, skim of representative files, cross-check against the 13 phases in `CLAUDE.md`. ~80K tokens, ~120 tool calls.
- **evva audit**: grep + file reads across `internal/`, checking imports for boundary leaks, sizes for test-coverage gaps, and named symbols mentioned in `CLAUDE.md` (e.g. `mainTaskPlanningSection`, `nameTaskCreate`) for incomplete phase cleanup. ~85K tokens, ~110 tool calls.

Neither pass was exhaustive — file reads have a window, and the audit prioritized verified findings over speculative ones. Treat sizes as rough; treat findings as starting points for the actual refactor PRs.
