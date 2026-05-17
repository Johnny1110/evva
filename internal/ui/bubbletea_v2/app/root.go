// Package app is the v2 TUI's top-level tea.Model. It stays thin on
// purpose — focus stack, layout engine, and msg dispatch live here;
// every visual concern lives in a sibling component package.
package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnny1110/evva/internal/ui"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/transcript"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/events"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/theme"
	"github.com/johnny1110/evva/pkg/banner"
)

// defaultGreeting is the welcome line rendered inside the banner box
// on startup. Short, gestures at next action, sets the tone.
const defaultGreeting = "// neural link established — what shall we build, ʘᴥʘ?"

// App is the v2 root model. M3 mounts a transcript + viewport so
// the welcome banner renders. Subsequent milestones grow this file
// without restructuring:
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

	theme      *theme.Theme
	transcript *transcript.Transcript
	view       *transcript.View

	startedAt time.Time
}

// New builds a fresh App. The program reference is wired in
// afterwards (tea.NewProgram needs the model before the model can
// know about the program).
func New(evvaHome string) *App {
	th := theme.Default()
	tr := transcript.New()
	tr.SetTheme(th)
	tr.SetBanner(transcript.BannerSpec{
		Art:      banner.Load(evvaHome),
		Greeting: defaultGreeting,
	})
	v := transcript.NewView(tr)

	return &App{
		evvaHome:   evvaHome,
		theme:      th,
		transcript: tr,
		view:       v,
		startedAt:  time.Now(),
	}
}

// SetProgram lets the package-level UI hand the model the program
// reference so future milestones can dispatch async commands via
// program.Send.
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
}

// Attach hands the model the agent controller and re-renders the
// banner with the controller's metadata (agent id, model, started
// at). Called once by the host between New and Run.
func (a *App) Attach(c ui.Controller) {
	a.controller = c
	a.refreshBanner()
	a.view.MarkDirty()
}

// refreshBanner repopulates the welcome banner with controller
// metadata. Safe to invoke before any width is known — BannerBlock
// gracefully degrades when ctx.Width is 0.
func (a *App) refreshBanner() {
	if a.controller == nil {
		return
	}
	id := a.controller.AgentID()
	if len(id) > 8 {
		id = id[:8]
	}
	a.transcript.SetBanner(transcript.BannerSpec{
		Art:      banner.Load(a.evvaHome),
		Greeting: defaultGreeting,
		Info: []transcript.BannerInfo{
			{Label: "agent", Value: id},
			{Label: "model", Value: a.controller.Model()},
			{Label: "started", Value: a.startedAt.Format("2006-01-02 15:04:05")},
		},
	})
}

// Init satisfies tea.Model. No initial command; later milestones
// return tea.Batch(spinnerTick, textareaBlink, ...).
func (a *App) Init() tea.Cmd { return nil }

// Update routes incoming messages. M3 handles:
//   - WindowSizeMsg → propagate to the viewport
//   - QuitMsg / Esc / Ctrl+C → exit
//   - AgentEventMsg → forward to the transcript ingestion
//   - tea.KeyMsg (other) → forward to the viewport for PgUp/PgDn etc.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.view.SetSize(msg.Width, msg.Height)
		return a, nil

	case events.QuitMsg:
		return a, tea.Quit

	case events.AgentEventMsg:
		if a.transcript.IngestEvent(msg.Event) {
			a.view.MarkDirty()
		}
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return a, tea.Quit
		}
		// Forward to viewport so PgUp/PgDn/Home/End scroll work.
		// M4 will route key events through a focus stack instead.
		return a, a.view.Update(msg)
	}
	return a, nil
}

// View returns the rendered scrollback. M2's placeholder is gone;
// the viewport now shows the real transcript (just the banner until
// M4 wires input).
func (a *App) View() string {
	return a.view.View()
}
