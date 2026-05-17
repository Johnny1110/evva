// Package agents renders the horizontal subagent chip strip that
// sits just above the input. Each in-flight (or recently completed)
// subagent appears as one bracketed chip:
//
//	‹⠋ explorer› ‹▶ writer› ‹✔ reviewer›
//
// Chips animate their leading glyph for active statuses (thinking,
// executing, draining, …); terminal statuses (ready_report,
// crushed) show their static glyph. Async subagents get a small
// superscript "ᵃ" so the user can see fire-and-forget vs. blocking.
//
// Returns "" when no subagents are tracked so the layout collapses
// the slot entirely.
package agents

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnny1110/evva/internal/tools/meta"
	"github.com/johnny1110/evva/internal/toolset"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/theme"
)

// agentChipMaxName caps the visible name length inside a chip so
// several agents can fit on one row. Names beyond this truncate to
// `chars-1` + "…".
const agentChipMaxName = 12

// Render returns the chip strip as a styled (possibly multi-line)
// string. width is the available column count; chips that don't
// fit wrap to a fresh row rather than truncating — losing
// visibility of a running agent is worse than spending an extra
// row.
//
// frame is the spinner index from the App's State; animated chips
// pick their glyph from theme.SpinnerFrame(frame).
func Render(ts *toolset.ToolState, width int, th *theme.Theme, frame int) string {
	if ts == nil || !ts.HasAgentGroupPanel() {
		return ""
	}
	rows := ts.AgentGroup().Snapshot()
	if len(rows) == 0 {
		return ""
	}
	if width < 1 {
		return ""
	}

	spacer := th.DimText.Render(" ")
	var lines []string
	var current strings.Builder
	currentWidth := 0
	for _, r := range rows {
		chip := renderChip(r, th, frame)
		chipWidth := lipgloss.Width(chip)
		// 1 col spacer between chips on the same row.
		needWidth := chipWidth
		if currentWidth > 0 {
			needWidth++
		}
		if currentWidth > 0 && currentWidth+needWidth > width {
			lines = append(lines, current.String())
			current.Reset()
			currentWidth = 0
		}
		if currentWidth > 0 {
			current.WriteString(spacer)
			currentWidth++
		}
		current.WriteString(chip)
		currentWidth += chipWidth
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
}

// renderChip formats one subagent as ‹glyph name›. Chevrons + name
// + glyph all share the agent's status color so the chip reads as
// one unit. Async subagents get a dim "ᵃ" superscript before the
// closing chevron.
func renderChip(r meta.SubagentSnapshot, th *theme.Theme, frame int) string {
	status := strings.ToLower(r.Status)
	glyph := renderStatusGlyph(status, th, frame)

	name := r.Name
	if name == "" {
		name = r.ID
	}
	if len(name) > agentChipMaxName {
		name = name[:agentChipMaxName-1] + "…"
	}

	c := chipColor(status, th)
	chev := lipgloss.NewStyle().Foreground(c).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(c)
	async := ""
	if r.Async {
		async = th.DimText.Render("ᵃ")
	}
	return chev.Render("‹") + glyph + " " + nameStyle.Render(name) + async + chev.Render("›")
}

// renderStatusGlyph picks the right symbol for a status string: the
// rotating spinner frame in the spinner color when the status is
// active, the static palette glyph otherwise.
func renderStatusGlyph(status string, th *theme.Theme, frame int) string {
	if style, ok := th.SpinnerStyle(status); ok {
		return style.Render(theme.SpinnerFrame(frame))
	}
	g := th.Glyph(status)
	return lipgloss.NewStyle().Foreground(g.Color).Render(g.Symbol)
}

// chipColor maps a subagent's lifecycle status to its chip color.
// Mirrors the status pill vocabulary so agent state reads
// consistently across the bottom of the UI.
//
// We extract the color via lipgloss.Color() from theme styles that
// already encode the right hue — keeps the palette private to the
// theme package.
func chipColor(status string, th *theme.Theme) lipgloss.Color {
	var c lipgloss.TerminalColor
	switch status {
	case "thinking", "texting":
		c = th.Thinking.GetForeground() // cool grey-blue cluster
	case "executing":
		c = th.ToolCall.GetForeground() // brown
	case "draining", "saving":
		c = th.Draining.GetForeground() // purple
	case "compacting", "max_iters":
		c = th.Compacting.GetForeground() // yellow
	case "ready_report", "idle":
		c = th.TasksDone.GetForeground() // green
	case "crushed", "interrupted":
		c = th.ErrorBanner.GetForeground() // red
	case "init":
		c = th.DimText.GetForeground() // muted
	default:
		c = th.ContextFill.GetForeground() // cyan
	}
	if col, ok := c.(lipgloss.Color); ok {
		return col
	}
	return lipgloss.Color("#7A7E94")
}
