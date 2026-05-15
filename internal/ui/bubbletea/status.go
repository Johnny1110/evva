package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnny1110/evva/internal/constant"
	"github.com/johnny1110/evva/internal/llm"
)

// runState enumerates the agent's high-level state from the UI's
// perspective. Drives status-bar text, input-disable logic, and the
// state color in the status pill.
//
// The "active" states (thinking / texting / executing / draining /
// compacting) all render with the rotating braille spinner so the
// user can tell at a glance that work is in flight; idle / paused /
// error use static glyphs.
type runState int

const (
	stateIdle runState = iota
	stateRunning    // generic "agent loop is alive between sub-phases"
	stateThinking   // model is generating reasoning tokens
	stateTexting    // model is generating response content tokens
	stateExecuting  // a tool call is in flight
	stateDraining   // pulling async subagent results back
	stateCompacting // micro/full session compaction running
	stateIterLimit
	stateError
)

func (s runState) String() string {
	switch s {
	case stateRunning:
		return "running"
	case stateThinking:
		return "thinking"
	case stateTexting:
		return "texting"
	case stateExecuting:
		return "executing"
	case stateDraining:
		return "draining"
	case stateCompacting:
		return "compacting"
	case stateIterLimit:
		return "paused"
	case stateError:
		return "error"
	default:
		return "ready"
	}
}

// isActive reports whether the state represents work-in-flight, i.e.
// the status pill should animate with the spinner rather than show a
// static dot.
func (s runState) isActive() bool {
	switch s {
	case stateRunning, stateThinking, stateTexting, stateExecuting,
		stateDraining, stateCompacting:
		return true
	}
	return false
}

// stateColor maps a runState onto the palette color used by the status
// pill. Vocabulary:
//   - green   → idle, work complete
//   - blue    → reasoning ("thinking" reads as cool, contemplative)
//   - cyan    → emitting content ("texting" — fluent output)
//   - orange  → tool execution (matches the orange ToolCall style)
//   - magenta → draining (subagent results coming in)
//   - yellow  → housekeeping (compacting, iter-limit)
//   - red     → fault
func stateColor(s runState) lipgloss.Color {
	switch s {
	case stateRunning:
		return paletteCyan
	case stateThinking:
		return paletteLightBlue
	case stateTexting:
		return paletteCyan
	case stateExecuting:
		return paletteOrange
	case stateDraining:
		return paletteMagenta
	case stateIterLimit, stateCompacting:
		return paletteYellow
	case stateError:
		return paletteRed
	default:
		return paletteGreen
	}
}

// statusBarInput bundles everything the bottom status bar renders so the
// callsite doesn't have to plumb half a dozen positional args.
type statusBarInput struct {
	Width        int
	Model        string
	Usage        llm.Usage
	State        runState
	Frame        int
	ContextUsed  int // tokens currently in the prompt (last turn's input)
	ContextLimit int // model's context window from constant.MODEL_CONTEXT_SIZE
}

// renderStatusBar formats the bottom one-liner. The leading column is
// the animated state pill — that is the highest-signal cell ("am I
// running, paused, errored?") so it sits first where the eye lands.
//
// Layout: `⠋ running  ·  evva  ·  ▸ model  ·  in X  out Y  ·  Context [████░ ] 39%`.
// Width pads the bar so it fills the terminal.
func renderStatusBar(in statusBarInput) string {
	parts := []string{
		renderStatePill(in.State, in.Frame),
		styles.StatusValue.Render("evva"),
		styles.StatusKey.Render(" " + in.Model),
		styles.StatusKey.Render("in ") + styles.StatusValue.Render(humanTokens(in.Usage.InputTokens)) +
			styles.StatusKey.Render("  out ") + styles.StatusValue.Render(humanTokens(in.Usage.OutputTokens)),
		renderContextBar(in.ContextUsed, in.ContextLimit),
	}
	body := strings.Join(parts, "  ·  ")
	return styles.StatusBar.Width(in.Width).Render(body)
}

// renderStatePill formats the state label, swapping in a spinner frame
// for the leading glyph whenever the agent is doing something active.
// Idle / paused / error render a static glyph so the user can tell at
// a glance whether work is in flight.
func renderStatePill(state runState, frame int) string {
	style := lipgloss.NewStyle().Foreground(stateColor(state)).Bold(true)
	var glyph string
	switch {
	case state.isActive():
		glyph = spinnerFrame(frame)
	case state == stateError:
		glyph = "✗"
	case state == stateIterLimit:
		glyph = "⏸"
	default:
		glyph = "●"
	}
	return style.Render(glyph + " " + state.String())
}

// renderContextBar produces a fixed-width progress bar showing how full
// the current prompt is relative to the model's context window. The bar
// reads `Context [████░░░░░░] 39%`. used==0 collapses gracefully to 0%.
func renderContextBar(used, limit int) string {
	const barWidth = 12
	pct := 0
	if limit > 0 {
		pct = used * 100 / limit
		if pct > 100 {
			pct = 100
		}
	}
	filled := pct * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	bar := styles.ContextFill.Render(strings.Repeat("█", filled)) +
		styles.ContextRail.Render(strings.Repeat("░", barWidth-filled))
	return styles.ContextBar.Render("Context ") +
		styles.StatusKey.Render("[") + bar + styles.StatusKey.Render("] ") +
		styles.StatusValue.Render(fmt.Sprintf("%d%%", pct))
}

// contextLimitFor returns the model's context window from the static
// table, or 0 when unknown — renderContextBar treats 0 as "no data" and
// shows 0%.
func contextLimitFor(model string) int {
	return constant.MODEL_CONTEXT_SIZE[constant.Model(model)]
}

// humanTokens formats a raw token count with a `k`/`m` suffix once it
// crosses the threshold. Keeps the status bar tight on long sessions.
func humanTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 10_000:
		return fmt.Sprintf("%dk", n/1_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
