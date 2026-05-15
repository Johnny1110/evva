package bubbletea

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// spinnerFrames is the braille-dot rotation used for any "ing" status
// (thinking, executing, draining, compacting). One frame per
// spinnerInterval; the model wraps via spinnerFrame(i).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerInterval is the wall-clock cadence at which the TUI advances
// the spinner. 100 ms matches the rate the user requested and is light
// enough for a TUI redraw budget.
const spinnerInterval = 100 * time.Millisecond

// spinnerFrame returns one frame of the rotation. The argument can be
// any non-negative tick counter; spinnerFrame handles wrapping.
func spinnerFrame(i int) string {
	if i < 0 {
		i = -i
	}
	return spinnerFrames[i%len(spinnerFrames)]
}

// activeSpinnerStatuses lists every status string that should render
// with a live spinner instead of a static glyph. Kept lowercase to
// match the lookup convention used by glyphFor.
var activeSpinnerStatuses = map[string]lipgloss.Style{
	"thinking":   styles.SpinThink,
	"in_progress": styles.SpinThink,
	"executing":  styles.SpinExec,
	"draining":   lipgloss.NewStyle().Foreground(paletteBlue).Bold(true),
	"compacting": lipgloss.NewStyle().Foreground(paletteYellow).Bold(true),
	"texting":    styles.SpinThink,
	"saving":     lipgloss.NewStyle().Foreground(paletteBlue).Bold(true),
}

// spinnerGlyphFor returns the spinner style for an "ing" status; the
// second return is false when the status is terminal (idle / done /
// crushed) so the caller can fall back to the static glyphFor table.
func spinnerGlyphFor(status string) (lipgloss.Style, bool) {
	st, ok := activeSpinnerStatuses[status]
	return st, ok
}

// statusGlyph pairs a status string (task or subagent) with the symbol +
// foreground color the TUI uses to render it. One source of truth for
// the panels, the transcript's all-done snapshot, and any future widget
// that wants to surface lifecycle state with consistent vocabulary.
type statusGlyph struct {
	Symbol string
	Color  lipgloss.Color
}

// statusSymbols is a flat map of every status string the TUI knows about.
// Task statuses (lowercase: "pending", "in_progress", "completed",
// "deleted") and agent statuses (constant.AgentStatus values: "init",
// "thinking", "executing", "draining", "compacting", "ready_report",
// "crushed", "max_iters", "idle", "interrupted", "saving", "shutdown",
// "texting") share the same table so consumers don't need to know which
// store the row came from.
//
// Keep the keys lowercase. Callers should normalize via strings.ToLower
// before lookup to be tolerant of future status-rename refactors.
var statusSymbols = map[string]statusGlyph{
	// task lifecycle
	"pending":     {Symbol: "☐", Color: paletteDim},
	"in_progress": {Symbol: "◐", Color: paletteYellow},
	"completed":   {Symbol: "☑", Color: paletteGreen},
	"deleted":     {Symbol: "·", Color: paletteDim},

	// subagent lifecycle (mirrors constant.AgentStatus values)
	"init":         {Symbol: "◌", Color: paletteDim},
	"thinking":     {Symbol: "◐", Color: paletteYellow},
	"executing":    {Symbol: "▶", Color: paletteCyan},
	"draining":     {Symbol: "◌", Color: paletteBlue},
	"compacting":   {Symbol: "↻", Color: paletteYellow},
	"ready_report": {Symbol: "✓", Color: paletteGreen},
	"crushed":      {Symbol: "✗", Color: paletteRed},
	"max_iters":    {Symbol: "⊘", Color: paletteYellow},
	"idle":         {Symbol: "✓", Color: paletteGreen},
	"interrupted":  {Symbol: "✗", Color: paletteRed},
	"saving":       {Symbol: "◌", Color: paletteBlue},
	"shutdown":     {Symbol: "✗", Color: paletteDim},
	"texting":      {Symbol: "◐", Color: paletteCyan},
}

// glyphFor returns the symbol + color for a status string. Unknown
// statuses get a neutral dim "·" — visible but not screaming for
// attention.
func glyphFor(status string) statusGlyph {
	if g, ok := statusSymbols[status]; ok {
		return g
	}
	return statusGlyph{Symbol: "·", Color: paletteDim}
}
