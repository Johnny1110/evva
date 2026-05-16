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

// statusGlyph pairs a status string (task or subagent) with the symbol
// + foreground color the TUI uses to render it. One source of truth
// for panels, transcript snapshots, and any future widget that wants
// consistent lifecycle vocabulary.
type statusGlyph struct {
	Symbol string
	Color  lipgloss.Color
}

// statusSymbols is the static-glyph table — used for non-active
// statuses. Active statuses (thinking / executing / draining /
// compacting) route through spinnerGlyphFor instead so they animate.
//
// HUD vocabulary: filled blocks / triangles / diamonds reads as
// "system state indicator" more than the soft circles a chat UI
// would use. ▶ for running, ◆ for steady, ✔ ✘ ⏸ for terminal states.
var statusSymbols = map[string]statusGlyph{
	// task lifecycle
	"pending":     {Symbol: "▢", Color: paletteMuted},
	"in_progress": {Symbol: "▶", Color: paletteYellow},
	"completed":   {Symbol: "▣", Color: paletteGreen},
	"deleted":     {Symbol: "·", Color: paletteDim},

	// subagent lifecycle (mirrors constant.AgentStatus values)
	"init":         {Symbol: "◇", Color: paletteMuted},
	"thinking":     {Symbol: "◆", Color: paletteLightBlue},
	"executing":    {Symbol: "▶", Color: paletteBrown},
	"draining":     {Symbol: "◈", Color: palettePurple},
	"compacting":   {Symbol: "↻", Color: paletteYellow},
	"ready_report": {Symbol: "✔", Color: paletteGreen},
	"crushed":      {Symbol: "✘", Color: paletteRed},
	"max_iters":    {Symbol: "⊘", Color: paletteYellow},
	"idle":         {Symbol: "✔", Color: paletteGreen},
	"interrupted":  {Symbol: "✘", Color: paletteRed},
	"saving":       {Symbol: "◇", Color: palettePurple},
	"shutdown":     {Symbol: "■", Color: paletteDim},
	"texting":      {Symbol: "◆", Color: paletteCyan},
}

// activeSpinnerStatuses lists every status string that should render
// with a live spinner instead of a static glyph. The mapped style
// carries the spinner's neon color.
var activeSpinnerStatuses = map[string]lipgloss.Style{
	"thinking":    styles.SpinThink,
	"in_progress": lipgloss.NewStyle().Foreground(paletteYellow).Bold(true),
	"executing":   styles.SpinExec,
	"draining":    lipgloss.NewStyle().Foreground(palettePurple).Bold(true),
	"compacting":  lipgloss.NewStyle().Foreground(paletteYellow).Bold(true),
	"texting":     lipgloss.NewStyle().Foreground(paletteCyan).Bold(true),
	"saving":      lipgloss.NewStyle().Foreground(palettePurple).Bold(true),
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

// spinnerGlyphFor returns the spinner style for an "ing" status; the
// second return is false when the status is terminal (idle / done /
// crushed) so the caller can fall back to the static glyphFor table.
func spinnerGlyphFor(status string) (lipgloss.Style, bool) {
	st, ok := activeSpinnerStatuses[status]
	return st, ok
}
