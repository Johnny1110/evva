package tasks

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/johnny1110/evva/internal/tools/task"
	"github.com/johnny1110/evva/internal/toolset"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/theme"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

// newTSWithTasks builds a fresh ToolState and pushes the given tasks
// into its TaskStore via Create (the store's only ingest path; the
// store assigns its own IDs internally so we re-apply the supplied
// Status afterwards via Update).
func newTSWithTasks(t *testing.T, items []task.Task) *toolset.ToolState {
	t.Helper()
	ts := toolset.NewToolState()
	store := ts.TaskStore()
	for _, it := range items {
		created := store.Create(it)
		// Create resets status to StatusPending; we may want a
		// different status for the test fixture.
		if it.Status != "" && it.Status != task.StatusPending {
			status := it.Status
			_, _, err := store.Update(created.ID, task.UpdatePatch{Status: &status})
			if err != nil {
				t.Fatalf("Update status %q: %v", it.Status, err)
			}
		}
	}
	return ts
}

func TestRenderEmpty(t *testing.T) {
	ts := toolset.NewToolState()
	if got := Render(ts, 80, theme.Default()); got != "" {
		t.Errorf("empty store should render empty, got %q", got)
	}
}

func TestRenderNilToolState(t *testing.T) {
	if got := Render(nil, 80, theme.Default()); got != "" {
		t.Errorf("nil ToolState should render empty, got %q", got)
	}
}

func TestRenderSkipsDeleted(t *testing.T) {
	ts := newTSWithTasks(t, []task.Task{
		{ID: "a", Subject: "still here", Status: task.StatusPending},
		{ID: "b", Subject: "should hide", Status: task.StatusDeleted},
	})
	out := Render(ts, 80, theme.Default())
	if !strings.Contains(out, "still here") {
		t.Errorf("missing live task: %q", out)
	}
	if strings.Contains(out, "should hide") {
		t.Errorf("deleted task leaked into render: %q", out)
	}
}

func TestRenderIncludesHeader(t *testing.T) {
	ts := newTSWithTasks(t, []task.Task{{ID: "1", Subject: "x", Status: task.StatusInProgress}})
	out := Render(ts, 80, theme.Default())
	if !strings.Contains(out, "TASKS") {
		t.Errorf("header missing TASKS label: %q", out)
	}
}

func TestRenderTruncatesLongSubject(t *testing.T) {
	long := strings.Repeat("x", 200)
	ts := newTSWithTasks(t, []task.Task{{ID: "1", Subject: long, Status: task.StatusPending}})
	out := Render(ts, 40, theme.Default())
	// The truncated suffix is the ellipsis rune; we just check that
	// the rendered row is shorter than the raw subject.
	if !strings.Contains(out, "…") {
		t.Errorf("expected ellipsis on truncated long subject: %q", out)
	}
}

// TestAllCompleted covers every branch of the predicate.
func TestAllCompleted(t *testing.T) {
	cases := []struct {
		name  string
		tasks []task.Task
		want  bool
	}{
		{"empty", nil, false},
		{"all deleted", []task.Task{{ID: "a", Subject: "x", Status: task.StatusDeleted}}, false},
		{"one pending", []task.Task{{ID: "a", Subject: "x", Status: task.StatusPending}}, false},
		{"all completed", []task.Task{
			{ID: "a", Subject: "x", Status: task.StatusCompleted},
			{ID: "b", Subject: "y", Status: task.StatusCompleted},
		}, true},
		{"completed plus deleted", []task.Task{
			{ID: "a", Subject: "x", Status: task.StatusCompleted},
			{ID: "b", Subject: "y", Status: task.StatusDeleted},
		}, true},
		{"one in progress", []task.Task{
			{ID: "a", Subject: "x", Status: task.StatusCompleted},
			{ID: "b", Subject: "y", Status: task.StatusInProgress},
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTSWithTasks(t, tc.tasks)
			if got := AllCompleted(ts); got != tc.want {
				t.Errorf("AllCompleted = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRenderCompleteSnapshot(t *testing.T) {
	ts := newTSWithTasks(t, []task.Task{
		{ID: "a", Subject: "ship the feature", Status: task.StatusCompleted},
		{ID: "b", Subject: "write docs", Status: task.StatusCompleted},
		{ID: "c", Subject: "old plan", Status: task.StatusDeleted}, // filtered
	})
	out := RenderCompleteSnapshot(ts, 80, theme.Default())
	if !strings.Contains(out, "TASKS COMPLETE") {
		t.Errorf("snapshot header missing: %q", out)
	}
	if !strings.Contains(out, "ship the feature") || !strings.Contains(out, "write docs") {
		t.Errorf("snapshot missing live tasks: %q", out)
	}
	if strings.Contains(out, "old plan") {
		t.Errorf("snapshot included deleted task: %q", out)
	}
}
