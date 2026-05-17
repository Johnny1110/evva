package bubbletea

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pendingCompact is the /compact chooser's in-flight state. Two rows:
// micro (instant, local elide) and full (one LLM call, ~5s). On Enter
// the chosen kind is dispatched through Controller.Compact in a
// goroutine so the chooser closes immediately and the transcript can
// paint the animated `<spinner> Compacting…` block.
type pendingCompact struct {
	choices  []compactChoice
	selected int
	// errMsg surfaces a dispatch failure (e.g. ErrRunInProgress).
	// Cleared on next navigation.
	errMsg string
}

// compactChoice is one row in the chooser. kind is the value passed to
// Controller.Compact; label and desc are the renderer's strings.
type compactChoice struct {
	kind  string
	label string
	desc  string
}

// compactChoices is the canonical option list. Micro first because it
// is instant and cheap; full sits below because the LLM call is more
// expensive and the user has to wait.
var compactChoices = []compactChoice{
	{kind: "micro", label: "Micro", desc: "elide older tool results · instant, no LLM call"},
	{kind: "full", label: "Full", desc: "ask the LLM to summarize the conversation · ~5s, replaces history with a brief"},
}

// openCompactPicker pushes the chooser into the pendingCompact slot.
// The cursor starts on the first row (Micro) so the cheap option is
// the default on Enter.
func (m *rootModel) openCompactPicker() {
	if m.controller == nil {
		m.hintText = "no controller attached"
		return
	}
	m.pendingCompact = &pendingCompact{
		choices:  compactChoices,
		selected: 0,
	}
}

func (m *rootModel) closeCompactPicker() {
	m.pendingCompact = nil
	m.layoutSizes()
}

// applyCompactChoice fires the chosen compaction. Full is launched in
// a goroutine so the TUI doesn't block on the LLM call — the agent
// emits KindCompacting (chooser sees it via the normal event sink),
// the transcript paints the animated row, and KindCompactingEnd
// removes it. Micro is also dispatched off-goroutine for symmetry; it
// returns near-instantly but keeping the same pattern simplifies the
// state machine.
func (m *rootModel) applyCompactChoice(c compactChoice) tea.Cmd {
	if m.controller == nil {
		return nil
	}
	// Use a fresh background context — manual compact is independent of
	// any active Run's ctx, and the agent's running guard already
	// prevents overlap.
	ctx := context.Background()
	ctrl := m.controller
	return func() tea.Msg {
		if err := ctrl.Compact(ctx, c.kind); err != nil {
			return compactDoneMsg{err: err}
		}
		return compactDoneMsg{}
	}
}

// compactDoneMsg signals the user-facing outcome of a manual compact.
// The chooser is already closed by the time this lands; the only
// effect is rendering a hint line on failure. Success is silent — the
// transcript end-event already conveys it.
type compactDoneMsg struct {
	err error
}

func (m *rootModel) handleCompactKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pc := m.pendingCompact
	if msg.Type == tea.KeyCtrlC {
		m.closeCompactPicker()
		m.cancelRunIfAny()
		return m, tea.Quit
	}
	switch msg.Type {
	case tea.KeyEsc:
		m.closeCompactPicker()
		return m, nil
	case tea.KeyUp:
		if pc.selected > 0 {
			pc.selected--
			pc.errMsg = ""
		}
		return m, nil
	case tea.KeyDown:
		if pc.selected < len(pc.choices)-1 {
			pc.selected++
			pc.errMsg = ""
		}
		return m, nil
	case tea.KeyEnter:
		cmd := m.applyCompactChoice(pc.choices[pc.selected])
		m.closeCompactPicker()
		return m, cmd
	}
	return m, nil
}

func (m *rootModel) compactPanel(width int) string {
	if m.pendingCompact == nil {
		return ""
	}
	innerWidth := width - 4
	if innerWidth < 30 {
		innerWidth = 30
	}
	return styles.InputBorder.Render(m.renderCompactList(innerWidth))
}

func (m *rootModel) renderCompactList(_ int) string {
	pc := m.pendingCompact

	var b strings.Builder
	b.WriteString(styles.PanelHeader.Render("▰ /COMPACT"))
	b.WriteByte('\n')
	b.WriteString(styles.DimText.Render(
		"Compaction reshapes the conversation to free context room. Micro is local and instant; Full asks the LLM for a summary brief.",
	))
	b.WriteString("\n\n")

	sel := lipgloss.NewStyle().Foreground(paletteCyan).Bold(true)
	dim := styles.DimText
	for i, c := range pc.choices {
		marker := "  "
		style := dim
		if i == pc.selected {
			marker = "▶ "
			style = sel
		}
		row := fmt.Sprintf("%s%-6s  %s", marker, c.label, c.desc)
		b.WriteString(style.Render(row))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	if pc.errMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(paletteMagenta).Render("✗ " + pc.errMsg))
		b.WriteByte('\n')
	}
	b.WriteString(styles.FooterHint.Render("[↑↓] navigate · [Enter] confirm · [Esc] cancel"))
	return strings.TrimRight(b.String(), "\n")
}
