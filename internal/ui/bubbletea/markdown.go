package bubbletea

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownRenderer wraps glamour with a width-aware constructor so the
// transcript can re-render content blocks when the terminal resizes.
//
// glamour's TermRenderer parses its style once at construction, so we
// cache one per width — recreate-on-resize keeps line wrapping in sync
// with the viewport without re-parsing styles on every chunk.
type markdownRenderer struct {
	width int
	term  *glamour.TermRenderer
}

// newMarkdownRenderer builds a renderer keyed to width. Falls back to a
// nil renderer (passthrough render) if glamour init fails — markdown is
// a nicety, not load-bearing.
func newMarkdownRenderer(width int) *markdownRenderer {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return &markdownRenderer{width: width}
	}
	return &markdownRenderer{width: width, term: r}
}

// Render returns the markdown-rendered version of s, falling back to
// the raw text if glamour is unavailable or the render errors out.
// Trailing newlines are trimmed so blocks join cleanly in the transcript.
func (r *markdownRenderer) Render(s string) string {
	if r == nil || r.term == nil {
		return s
	}
	out, err := r.term.Render(s)
	if err != nil {
		return s
	}
	return strings.TrimRight(out, "\n")
}
