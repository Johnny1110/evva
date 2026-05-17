// Package app is the v2 TUI's top-level tea.Model. It stays thin on
// purpose — focus stack, layout engine, and msg dispatch live here;
// every visual concern lives in a sibling component package.
package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnny1110/evva/internal/ui"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/events"
)

// App is the v2 root model. The skeleton (M1) boots clean, shows a
// placeholder, and quits on Esc / Ctrl+C. Subsequent milestones grow
// it without restructuring this file:
//   - M3 mounts the transcript component
//   - M4 mounts the input
//   - M5 mounts the status bar + run state
//   - M6 mounts tasks panel + agents strip
//   - M7 mounts slash panel + overlay focus stack
//   - M8 enables mouse capture + yank mode
//   - M9 mounts search overlay
//   - M10 mounts permission overlay
type App struct {
	evvaHome   string
	program    *tea.Program
	controller ui.Controller

	width  int
	height int
}

// New builds a fresh App. The program reference is wired in afterwards
// (the ctor pattern in v1 — tea.NewProgram needs the model before the
// model can know about the program).
func New(evvaHome string) *App {
	return &App{evvaHome: evvaHome}
}

// SetProgram lets the package-level UI hand the model the program
// reference so future milestones can dispatch async commands via
// program.Send. The skeleton doesn't use it yet.
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
}

// Attach hands the model the agent controller. Future milestones use
// this to call Run/Continue, read Session(), read ToolState(), etc.
func (a *App) Attach(c ui.Controller) {
	a.controller = c
}

// Init satisfies tea.Model. No initial command in the skeleton; later
// milestones return tea.Batch(spinnerTick, textareaBlink, ...).
func (a *App) Init() tea.Cmd { return nil }

// Update routes incoming messages. The skeleton handles only window
// resize, the quit pathways, and silently absorbs agent events (the
// event router lands in M3).
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil

	case events.QuitMsg:
		return a, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return a, tea.Quit
		}

	case events.AgentEventMsg:
		// Skeleton drops agent events on the floor. M3's dispatch
		// router fans them out to components.
		return a, nil
	}
	return a, nil
}

// View renders the placeholder. M2+ replaces this with the layout
// engine's Compose() output.
func (a *App) View() string {
	var b strings.Builder
	b.WriteString("bubbletea_v2 — skeleton (M1)\n")
	b.WriteString("\n")
	fmt.Fprintf(&b, "size: %dx%d\n", a.width, a.height)
	if a.controller != nil {
		fmt.Fprintf(&b, "agent: %s · model: %s\n", a.controller.AgentID(), a.controller.Model())
	} else {
		b.WriteString("agent: (not attached)\n")
	}
	b.WriteString("\n")
	b.WriteString("press Esc or Ctrl+C to quit")
	return b.String()
}
