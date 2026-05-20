# EVVAgent (evva)

A ReAct coding agent in your terminal. Multi-provider LLM, parallel tool dispatch, async sub-agents, swappable UI.

---

## What is EVVAgent?

`evva` runs a tool-using LLM agent in your terminal. It speaks Anthropic Claude, DeepSeek, OpenAI, and Ollama through one `llm.Client` interface; dispatches multiple tool calls per turn in parallel; tracks tasks and sub-agents through an observable store; and renders into a bubbletea TUI or a plain-text CLI sink.

The architecture is small on purpose — adding a new LLM provider, panel, or UI implementation is roughly one package each.

---

## Install

```bash
git clone https://github.com/johnny1110/evva
cd evva
make install
```

Default install target is `$GOBIN` (or `$GOPATH/bin` when `GOBIN` is unset) — usually already on a Go developer's `PATH`. The `make install` output tells you whether to add it.

Override the location if you want it elsewhere:

```bash
sudo make install PREFIX=/usr/local/bin     # system-wide
make install PREFIX=$HOME/.local/bin        # user-local
```

Verify:

```bash
which evva
evva --help-ish    # any flag triggers the usage line
```

Uninstall removes only the binary; your `~/.evva/` config is preserved:

```bash
make uninstall
```

---

## First run

Just type `evva` from any directory. On the first launch evva auto-creates:

```
~/.evva/
├── config/
│   └── evva-config.yml      # user-tunable settings (auto-created with defaults)
└── skills/                  # optional skill scripts (your own)
```

A one-line stderr notice fires the first time only:

```
evva: wrote new config to ~/.evva/config/evva-config.yml — fill in your API keys to use cloud providers.
```

`~/.evva/.env` is **optional**. If you want to override deployment knobs (`LOG_LEVEL`, `LOG_DIR`, `APP_ENV`, `LOG_FORMAT`, `SKILLS_DIR`, `USER_PROFILE`), create it; otherwise the built-in defaults apply.

### Adding an API key

Two ways:

1. **From inside the TUI:** type `/config`, navigate to `<provider>.api_key`, press Enter, paste your key, press Enter again. Saved immediately.
2. **By hand:** open `~/.evva/config/evva-config.yml` and fill in `providers.<provider>.api_key`.

Cloud providers (Anthropic, DeepSeek, OpenAI) need a key; Ollama is local and key-less.

---

## User Guide

Full usage documentation covering the TUI interface, slash commands, keybindings, yank mode, the permission system, sub-agents, and all configuration options:

- [English](docs/user-guide/en/user-guide.md)
- [正體中文](docs/user-guide/zh-tw/user-guide.md)

---

## Configuration

### `~/.evva/config/evva-config.yml`

User-tunable settings. Created automatically on first launch. Edit live via `/config` in the TUI, or by hand:

```yaml
# Agent loop
max_iterations: 30
max_tokens: 4096
auto_compact_threshold: 0.8
display_thinking: true

# Default model used at startup (overwritten by /model swap)
default_provider: deepseek
default_model: deepseek-v4-pro

# Permission stance at startup. Cycle at runtime with Shift+Tab; -permission-mode CLI flag overrides.
permission_mode: default     # default | accept_edits | plan | bypass

# Web tooling
fetch_max_bytes: 100000
tavily_api_key: ""

# Per-provider credentials. Empty api_url falls back to the constant's default.
providers:
  anthropic: { api_key: "", api_url: "" }
  deepseek:  { api_key: "", api_url: "" }
  openai:    { api_key: "", api_url: "" }
  ollama:    { api_url: "" }
```

### `.env` (optional)

Place in your working directory or at `~/.evva/.env`. Only used for deployment / logging knobs — never user preferences:

```bash
APP_ENV=dev            # dev | prod
LOG_LEVEL=info         # debug | info | warn | error
LOG_FORMAT=text        # text | json
LOG_DIR=               # empty → stdout; path → write log files there
SKILLS_DIR=skills      # subpath under ~/.evva/
USER_PROFILE=user_profile.md
```

### CLI flags

```bash
evva                                # interactive TUI (when stdout is a TTY)
evva -temp 0.7                      # sampling temperature (default unset)
evva -max-tokens 2048               # per-completion output cap (overrides YAML)
evva -max-iters 40                  # loop iteration cap (overrides YAML)
evva -permission-mode=plan          # boot in plan mode (read-only; see "Permission modes")
evva -permission-mode=bypass        # boot with the gate disabled
evva -no-hooks                      # disable user-authored hooks for this run (see "Hooks")
evva -no-tui "explain loop.go"      # one-shot plain-text mode
echo "list files in /tmp" | evva -no-tui   # piped prompt
```

---

## Hooks

Hooks are user-authored shell commands or HTTP webhooks that fire at six lifecycle moments — before a tool call, after a tool call, when you submit a prompt, on session start, when the agent finishes, or on side-channel notifications (errors, iteration limit, approval-needed). They let you wire validation, auto-format, audit logging, or "block known-bad commands" logic into evva without forking the source.

Hooks **compose** with the permission system: a `PreToolUse` hook runs *before* the permission gate and can override the gate's decision (allow / deny / ask) or mutate the tool's input before the gate sees it. Hooks are honored in every permission mode, including `bypass` — they are user-authored, so the bypass-mode "I know what I'm doing" stance does not apply.

### Events

| Event | When | Can block? | Payload extras |
|---|---|---|---|
| `SessionStart` | First `Run()` of an agent (incl. subagents) | No | `source` (`startup`), `model` |
| `UserPromptSubmit` | Each user prompt, before it lands in the session | Yes — drops the prompt | `prompt` |
| `PreToolUse` | Each tool call, before the permission gate | Yes; can also mutate input / override gate | `tool_name`, `tool_input`, `tool_use_id` |
| `PostToolUse` | After the tool returns (success or error) | No — can append `additionalContext` only | `tool_name`, `tool_input`, `tool_response`, `is_error`, `tool_use_id` |
| `Stop` | Main agent reaches a terminal turn (no more tool calls) | Yes — re-enters the loop once with a synthetic user message | `stop_hook_active`, `last_assistant_message` |
| `Notification` | Iter-limit, errors, approval-needed | No (fire-and-forget) | `message`, `title`, `notification_type` |

Every hook payload also carries a base envelope: `session_id`, `cwd`, `permission_mode`, `agent_id` / `agent_type` (subagents only), and `hook_event_name`.

### Storage

Hooks bind in JSON files, layered by scope:

- `./.evva/settings.json` — **project** scope, in your repo.
- `~/.evva/settings.json` — **user** scope, applies everywhere.

Both files are read on every startup. Project hooks fire **before** user hooks for the same event, and a project hook returning `continue: false` short-circuits the user hook chain.

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "write",
        "hooks": [
          { "type": "command", "command": "bash ~/.evva/hooks/validate-write.sh", "timeout": 5 }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "write|edit",
        "hooks": [
          { "type": "command", "command": "bash ~/.evva/hooks/gofmt-on-save.sh" }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          { "type": "http", "url": "https://hooks.slack.com/services/...", "async": true }
        ]
      }
    ]
  }
}
```

The `matcher` field is a tool-name glob (uses doublestar syntax). Empty matcher or omitted = matches all tools / all firings of that event.

### Hook contract

#### `type: "command"` (shell subprocess)

evva invokes `/bin/sh -c "<command>"` with:
- **stdin** — the event payload as JSON.
- **env** — your current env plus `EVVA_PROJECT_DIR`, `EVVA_SESSION_ID`, `EVVA_AGENT_ID`.
- **stdout** — optional JSON decision (parsed for PreToolUse / Stop / UserPromptSubmit / SessionStart; ignored for PostToolUse / Notification beyond `additionalContext`).
- **exit codes**:
  - `0` — success; stdout parsed as JSON.
  - `1` — non-blocking error; logged, the hook chain continues.
  - `2` — block (legacy/non-JSON shortcut); stderr is used as the block reason.

The JSON decision shape:

```json
{
  "continue": true,
  "decision": "approve" | "block" | null,
  "reason": "string",
  "systemMessage": "string",
  "hookSpecificOutput": {
    "permissionDecision": "allow" | "deny" | "ask",
    "permissionDecisionReason": "string",
    "updatedInput": { ... },
    "additionalContext": "string",
    "initialUserMessage": "string"
  }
}
```

Per-event field semantics:

- **PreToolUse**: `permissionDecision` overrides the gate. `updatedInput` replaces `tool_input` before the gate / tool runs. `additionalContext` is currently ignored for PreToolUse — append context via PostToolUse instead.
- **PostToolUse**: only `additionalContext` matters. It's appended to the tool's result content for the LLM's next turn.
- **UserPromptSubmit**: `additionalContext` is appended to the prompt. `decision: "block"` or `continue: false` drops the prompt entirely.
- **SessionStart**: `initialUserMessage` is prepended to the session as a synthetic user message; `additionalContext` is appended to the first real prompt.
- **Stop**: `decision: "block"` / `continue: false` re-enters the loop once, with `reason` as a synthetic user message. The re-entry pass sets `stop_hook_active: true` so a second block is ignored (prevents infinite loops).
- **Notification**: stdout ignored — always fire-and-forget.

#### `type: "http"` (HTTP webhook)

evva sends an HTTP request with the JSON payload as body. Defaults to `POST`, `Content-Type: application/json`, 10-second timeout, and `async: true` (fire-and-forget). Non-2xx in sync mode is logged as a non-blocking error.

```json
{
  "type": "http",
  "url": "https://example.com/evva-hook",
  "method": "POST",
  "headers": { "Authorization": "Bearer xxx" },
  "timeout": 5,
  "async": true
}
```

### Cookbook

**Auto-format Go files on write.** Place at `~/.evva/hooks/gofmt-on-save.sh`:

```bash
#!/usr/bin/env bash
path=$(jq -r '.tool_input.file_path' <&0 2>/dev/null)
[[ "$path" == *.go ]] && gofmt -w "$path"
exit 0
```

Then in `~/.evva/settings.json`:
```json
{
  "hooks": {
    "PostToolUse": [
      { "matcher": "write|edit", "hooks": [
        { "type": "command", "command": "bash ~/.evva/hooks/gofmt-on-save.sh" }
      ]}
    ]
  }
}
```

**Block writes outside the repo root.** Place at `./.evva/hooks/sandbox-write.sh`:

```bash
#!/usr/bin/env bash
path=$(jq -r '.tool_input.file_path' <&0)
if [[ "$path" != "$EVVA_PROJECT_DIR"* ]]; then
  cat <<EOF
{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"write outside repo root"}}
EOF
fi
exit 0
```

```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "write", "hooks": [
        { "type": "command", "command": "bash ./.evva/hooks/sandbox-write.sh", "timeout": 2 }
      ]}
    ]
  }
}
```

**Slack ping on iteration-limit.** No shell needed:

```json
{
  "hooks": {
    "Notification": [
      { "hooks": [
        {
          "type": "http",
          "url": "https://hooks.slack.com/services/T0/B0/XXX",
          "headers": { "Content-Type": "application/json" },
          "async": true
        }
      ]}
    ]
  }
}
```

### Disabling

- `-no-hooks` CLI flag — skips loading entirely for one run (settings.json still on disk, just ignored).
- Delete or move `settings.json` files — no hooks loaded.
- Leave the `"hooks"` block empty — `{"hooks": {}}` is valid and produces no firings.

Bad entries (invalid JSON, unknown event name, malformed URL, out-of-range timeout) are surfaced as warnings on stderr at startup and the rest of the file still loads.

---

## Features

**Agent loop**
- ReAct-style: LLM call → parallel tool dispatch → tool results → repeat.
- Multiple `tool_use` blocks per turn, executed concurrently.
- Iteration cap surfaces as a pausable state.
- Cancellable via `ctx`; Esc / Ctrl+C honored end-to-end.

**LLM providers**
- Anthropic Claude (extended thinking + cryptographic signature round-trip).
- DeepSeek (OpenAI-compatible chat, reasoning_content echoed back).
- OpenAI.
- Ollama (local).
- Per-provider option pattern (`WithTemperature`, `WithEffort`, ...).

**Tools**
- File system: `read_file`, `write_file`, `edit_file` — strict-absolute paths, structured `*FileDiff` metadata for diff rendering.
- Shell: `bash`, `grep`, `tree`.
- Tasks (six tools sharing one observable `*task.Store`).
- Meta: `agent` (sub-agents), `tool_search` (lazy schema loading), `skill`, `schedule_wakeup`.
- Plus stubs for `web_*`, `cron_*`, `notebook_edit`, `monitor`, `mode`, `ux`.

**Sub-agents**
- `explore` (read-only) and `general-purpose` presets.
- Sync mode (parent blocks) and async mode (parent continues, result lands on next iteration).
- Two-layer hierarchy: sub-agents can't spawn sub-agents.

**Observable store framework** (`internal/observable`)
- One pub/sub primitive any store can embed. Adding a new panel costs zero edits to the agent or event packages.

**Swappable UI** (`internal/ui`)
- Narrow `UI` and `Controller` interfaces. Reference bubbletea TUI under `internal/ui/bubbletea/`. `-no-tui` falls back to a plain CLI sink.

**Streaming completions** (chunked text + thinking).

**2-level compaction**
- micro: compress tool-result blocks when context budget approaches threshold.
- full: summarize the whole session into a single assistant brief.

---

## Project structure

```
evva/
├── cmd/evva/                  # CLI entry point — wires agent + UI
├── configs/                   # config loading (.env + YAML)
├── docs/                      # design notes, tool docs, system prompts
├── internal/
│   ├── agent/                 # agent loop, profiles, spawn
│   │   ├── event/             # event types + sink contract
│   │   └── sysprompt/         # system prompt builder
│   ├── constant/              # provider / model / status enums
│   ├── llm/                   # llm.Client interface + shared params
│   │   ├── claude/  deepseek/  ollama/
│   ├── llmfactory/            # provider factory keyed by constant
│   ├── logger/                # structured slog wrapper + pretty fmt
│   ├── observable/            # pub/sub framework for stores
│   ├── session/               # conversation history + cumulative usage
│   ├── tools/                 # tool interface (Name/Schema/Execute)
│   │   ├── cron/  dev/  fs/  meta/  mode/  monitor/  notebook/
│   │   ├── shell/ task/ ux/   web/
│   ├── toolset/               # tool catalog + ToolState registry
│   └── ui/                    # UI plugin contract
│       └── bubbletea/         # reference TUI implementation
├── log/                       # per-agent runtime logs (gitignored)
├── pkg/common/                # small shared utilities (UUID, ...)
└── scripts/                   # demo / dev scripts
```

Key boundaries:
- `agent` knows about `event.Sink`, never about a concrete UI.
- `tools/*` packages produce `tools.Result` (text + opaque `Metadata`); the UI type-asserts on `Metadata` to render structured payloads.
- `observable` has no dependencies on agent or UI.
- `ui` defines two narrow interfaces; implementations live under it.

---

## Roadmap

### Planned
- **Multimodal Read**: images, PDFs (with `pages` range), Jupyter notebooks.
- **Overwrite diffs**: proper Myers/Hunt-McIlroy diff for `write_file` overwrites.
- **Per-agent LLM**: sub-agent can use a different provider than its parent.
- **Veronica space**: long-running local sandbox service on `:8080`.
- **Web UI**: a second `UI` implementation served over WebSocket.
- **Session persistence**: `/resume` to reload a session snapshot.

### Known limitations
- Sub-agent hierarchy is exactly two layers (no nested spawning).
- Token counts depend on provider reporting — Ollama only reports prompt / eval, not cache or reasoning splits.
- The TUI transcript grows unbounded in a long session; compaction is on the list above.

---

## License

See [LICENSE](LICENSE).
