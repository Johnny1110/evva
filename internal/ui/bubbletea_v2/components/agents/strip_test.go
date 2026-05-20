package agents

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/johnny1110/evva/pkg/constant"
	"github.com/johnny1110/evva/internal/tools/meta"
	"github.com/johnny1110/evva/internal/toolset"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/theme"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

// stripANSI — local helper for assertion-only use.
func stripANSI(s string) string {
	var b strings.Builder
	skip := false
	for _, r := range s {
		if r == 0x1b {
			skip = true
			continue
		}
		if skip {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '\x07' {
				skip = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// newTSWithAgents materialises the panel and seeds each snapshot
// via SpawnGroup.Add + Status (the store's actual ingest path).
// Status strings must match constant.AgentStatus values so the
// store accepts them.
func newTSWithAgents(t *testing.T, snaps []meta.SubagentSnapshot) *toolset.ToolState {
	t.Helper()
	ts := toolset.NewToolState()
	g := ts.AgentGroup()
	for _, s := range snaps {
		g.Add(s.Name, s.ID, s.Type, s.JobDesc, s.Async)
		if s.Status != "" {
			g.Status(s.ID, constant.AgentStatus(s.Status))
		}
	}
	return ts
}

func TestRenderEmpty(t *testing.T) {
	ts := toolset.NewToolState()
	// HasAgentGroupPanel is false until AgentGroup() is materialised.
	if got := Render(ts, 80, theme.Default(), 0); got != "" {
		t.Errorf("nil group should render empty, got %q", got)
	}
}

func TestRenderSingleChip(t *testing.T) {
	ts := newTSWithAgents(t, []meta.SubagentSnapshot{
		{ID: "ag-1", Name: "explorer", Type: "explore", Status: "thinking"},
	})
	out := Render(ts, 80, theme.Default(), 0)
	plain := stripANSI(out)
	if !strings.Contains(plain, "explorer") {
		t.Errorf("chip should include agent name: %q", plain)
	}
	if !strings.Contains(plain, "‹") || !strings.Contains(plain, "›") {
		t.Errorf("chip should be bracketed with chevrons: %q", plain)
	}
}

func TestRenderAsyncMarker(t *testing.T) {
	ts := newTSWithAgents(t, []meta.SubagentSnapshot{
		{ID: "ag-1", Name: "bg-job", Status: "executing", Async: true},
	})
	out := stripANSI(Render(ts, 80, theme.Default(), 0))
	if !strings.Contains(out, "ᵃ") {
		t.Errorf("async chip should include 'ᵃ' marker: %q", out)
	}
}

func TestRenderTruncatesLongName(t *testing.T) {
	long := strings.Repeat("a", 50)
	ts := newTSWithAgents(t, []meta.SubagentSnapshot{
		{ID: "ag-1", Name: long, Status: "thinking"},
	})
	out := stripANSI(Render(ts, 80, theme.Default(), 0))
	if strings.Contains(out, long) {
		t.Errorf("long name should be truncated, got: %q", out)
	}
	if !strings.Contains(out, "…") {
		t.Errorf("truncated name should end with '…': %q", out)
	}
}

func TestRenderWrapsToMultipleLines(t *testing.T) {
	// 6 agents at ~16 cols each (chip + spacer) easily overflow 60.
	snaps := []meta.SubagentSnapshot{
		{ID: "a", Name: "one", Status: "thinking"},
		{ID: "b", Name: "two", Status: "executing"},
		{ID: "c", Name: "three", Status: "draining"},
		{ID: "d", Name: "four", Status: "ready_report"},
		{ID: "e", Name: "five", Status: "crushed"},
		{ID: "f", Name: "six", Status: "init"},
	}
	ts := newTSWithAgents(t, snaps)
	out := Render(ts, 30, theme.Default(), 0)
	if strings.Count(out, "\n") < 1 {
		t.Errorf("expected at least one line wrap at width 30, got:\n%q", out)
	}
}

func TestRenderSpinnerFrameAdvances(t *testing.T) {
	ts := newTSWithAgents(t, []meta.SubagentSnapshot{
		{ID: "ag-1", Name: "spin", Status: "thinking"}, // active → animates
	})
	frame0 := Render(ts, 80, theme.Default(), 0)
	frame3 := Render(ts, 80, theme.Default(), 3)
	if frame0 == frame3 {
		t.Errorf("frame change should alter animated chip: frame0=%q frame3=%q", frame0, frame3)
	}
}
