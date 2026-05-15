package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnny1110/evva/internal/tools/task"
	"github.com/johnny1110/evva/internal/toolset"
)

// subagentPanelMinWidth is the floor for the left subagent column.
// Kept tight — most subagent names are short and the transcript
// benefits from every column. Names longer than 8 characters expand
// the panel dynamically via subagentPanelWidth.
const subagentPanelMinWidth = 12

// renderTaskPanel returns the bottom task panel string, or "" when no
// non-deleted tasks remain. Each row prefixes a status glyph in the
// shared palette so the visual vocabulary matches the subagent panel
// and the all-tasks-complete transcript snapshot.
//
// Width is the available column count; rows are truncated to fit.
func renderTaskPanel(ts *toolset.ToolState, width int) string {
	if ts == nil {
		return ""
	}
	store := ts.TaskStore()
	if store == nil {
		return ""
	}
	all := store.List()
	rows := make([]task.Task, 0, len(all))
	for _, t := range all {
		if t.Status == task.StatusDeleted {
			continue
		}
		rows = append(rows, t)
	}
	if len(rows) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.PanelHeader.Render("Tasks"))
	b.WriteByte('\n')
	for _, t := range rows {
		b.WriteString(renderTaskRow(t, width))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderTaskRow formats one task with its status glyph + subject.
// Subjects longer than the available width are truncated; the glyph
// stays visible so the user can always read progress at a glance.
func renderTaskRow(t task.Task, width int) string {
	g := glyphFor(string(t.Status))
	subject := t.Subject
	// Reserve room for "  G  " (2 spaces + symbol + 2 spaces).
	body := fmt.Sprintf("  %s  %s", styled(g.Symbol, g.Color), subject)
	maxLen := width - 6
	if maxLen > 0 && len(subject) > maxLen {
		body = fmt.Sprintf("  %s  %s", styled(g.Symbol, g.Color), truncate(subject, maxLen))
	}
	return body
}

// AllTasksCompleted reports whether every non-deleted task in the store
// has reached StatusCompleted. The TUI watches this on every task
// KindStoreUpdate to decide when to auto-fold the panel.
//
// Returns false on an empty/all-deleted store so a fresh store doesn't
// trigger a phantom "tasks complete" snapshot.
func AllTasksCompleted(ts *toolset.ToolState) bool {
	if ts == nil {
		return false
	}
	store := ts.TaskStore()
	if store == nil {
		return false
	}
	all := store.List()
	any := false
	for _, t := range all {
		if t.Status == task.StatusDeleted {
			continue
		}
		any = true
		if t.Status != task.StatusCompleted {
			return false
		}
	}
	return any
}

// renderTasksCompleteSnapshot renders the green "all tasks complete"
// block that gets folded into the transcript when the panel auto-clears.
// Mirrors renderTaskPanel's row shape but uses the green header style
// so the snapshot reads as a definite "done" event.
func renderTasksCompleteSnapshot(ts *toolset.ToolState, width int) string {
	if ts == nil {
		return ""
	}
	store := ts.TaskStore()
	if store == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(styles.TasksDone.Render("✓ Tasks complete"))
	for _, t := range store.List() {
		if t.Status == task.StatusDeleted {
			continue
		}
		b.WriteByte('\n')
		b.WriteString(renderTaskRow(t, width))
	}
	return b.String()
}

// renderSubagentPanel returns the LEFT-column subagent panel string,
// or "" when no subagents are tracked. Each row is one subagent: the
// status glyph + name + an "a" superscript when async.
//
// frame is the current spinner index; active "ing" statuses render the
// matching braille-dot frame in their status color so the panel shows
// motion as the agent works. Terminal statuses (idle, crushed, etc)
// keep their static glyph from the shared symbols table.
//
// The panel is intentionally narrow so the transcript keeps the bulk
// of the screen width. Rows that overflow the column get truncated;
// the status glyph and short name take priority.
func renderSubagentPanel(ts *toolset.ToolState, width, frame int) string {
	if ts == nil || !ts.HasAgentGroupPanel() {
		return ""
	}
	rows := ts.AgentGroup().Snapshot()
	if len(rows) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.PanelHeader.Render("Subagents"))
	b.WriteByte('\n')
	for _, r := range rows {
		status := strings.ToLower(r.Status)
		glyph := renderStatusGlyph(status, frame)
		name := r.Name
		if name == "" {
			name = r.ID
		}
		async := ""
		if r.Async {
			async = styles.DimText.Render(" a")
		}
		maxName := width - 4
		if r.Async {
			maxName -= 2
		}
		if maxName > 0 && len(name) > maxName {
			name = truncate(name, maxName)
		}
		row := fmt.Sprintf(" %s %s%s", glyph, styles.PanelRow.Render(name), async)
		b.WriteString(row)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderStatusGlyph picks the right symbol for a status string: the
// spinner frame in the spinner color when the status is active, the
// static palette glyph otherwise. Centralizing here keeps panel rows
// and the all-tasks-complete snapshot visually in sync.
func renderStatusGlyph(status string, frame int) string {
	if style, ok := spinnerGlyphFor(status); ok {
		return style.Render(spinnerFrame(frame))
	}
	g := glyphFor(status)
	return styled(g.Symbol, g.Color)
}

// subagentPanelWidth computes the column width the subagent panel
// should occupy: the minimum, expanded if any name is too long to fit.
// Returns 0 when the panel is collapsed (no rows) so the caller can
// give the column's screen real estate back to the transcript.
func subagentPanelWidth(ts *toolset.ToolState) int {
	if ts == nil || !ts.HasAgentGroupPanel() {
		return 0
	}
	rows := ts.AgentGroup().Snapshot()
	if len(rows) == 0 {
		return 0
	}
	width := subagentPanelMinWidth
	for _, r := range rows {
		name := r.Name
		if name == "" {
			name = r.ID
		}
		// 4 cols overhead: leading space, glyph, space, trailing slack
		need := len(name) + 4
		if r.Async {
			need += 2
		}
		if need > width {
			width = need
		}
	}
	return width
}

// styled wraps text in a foreground color without spinning up a fresh
// lipgloss.Style for every cell — keeps panel rendering cheap on
// frequent re-renders.
func styled(s string, c lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}
