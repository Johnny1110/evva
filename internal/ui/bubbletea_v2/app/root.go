// Package app is the v2 TUI's top-level tea.Model. It stays thin on
// purpose — focus stack, layout engine, and msg dispatch live here;
// every visual concern lives in a sibling component package.
package app

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnny1110/evva/internal/agent/event"
	"github.com/johnny1110/evva/internal/llm"
	"github.com/johnny1110/evva/internal/tools/task"
	"github.com/johnny1110/evva/internal/ui"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/agents"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/input"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/status"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/tasks"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/components/transcript"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/events"
	"github.com/johnny1110/evva/internal/ui/bubbletea_v2/theme"
	"github.com/johnny1110/evva/pkg/banner"
)

// defaultGreeting is the welcome line rendered inside the banner box
// on startup.
const defaultGreeting = "// neural link established — what shall we build, ʘᴥʘ?"

// App is the v2 root model. M5 adds the bottom HUD (status bar +
// contextual hint), the run-state machine, and the spinner tick.
// The "running" bool from M4 is gone — the State machine owns
// lifecycle; the cancel func still lives here because it's the App
// that drives the goroutine.
type App struct {
	evvaHome   string
	program    *tea.Program
	controller ui.Controller

	width  int
	height int

	theme      *theme.Theme
	transcript *transcript.Transcript
	view       *transcript.View
	input      *input.Input
	status     *status.StatusBar
	state      *status.State

	// runCancel is the cancel func for the in-flight Run, set in
	// startRun and cleared in handleRunDone. Used by the Esc /
	// Ctrl+C handlers to interrupt mid-flight.
	runCancel context.CancelFunc
	// interrupted captures the "user pressed Esc" signal so the
	// RunDoneMsg handler can pick the "interrupted" hint instead
	// of "error: ...". Cleared on next OnSubmit.
	interrupted bool

	startedAt time.Time
}

// New builds a fresh App. The program reference is wired in
// afterwards.
func New(evvaHome string) *App {
	th := theme.Default()
	tr := transcript.New()
	tr.SetTheme(th)
	tr.SetBanner(transcript.BannerSpec{
		Art:      banner.Load(evvaHome),
		Greeting: defaultGreeting,
	})
	v := transcript.NewView(tr)
	in := input.New(th)
	st := status.NewState()
	bar := status.New(st)

	return &App{
		evvaHome:   evvaHome,
		theme:      th,
		transcript: tr,
		view:       v,
		input:      in,
		status:     bar,
		state:      st,
		startedAt:  time.Now(),
	}
}

// SetProgram lets the package-level UI hand the model the program
// reference. Used by the run goroutine to dispatch RunDoneMsg back
// to the bubbletea main loop.
func (a *App) SetProgram(p *tea.Program) { a.program = p }

// Attach hands the model the agent controller and re-renders the
// banner. Also primes the status bar with model + agent id and the
// initial context limit.
func (a *App) Attach(c ui.Controller) {
	a.controller = c
	a.refreshBanner()
	a.status.SetModel(c.Model())
	a.status.SetAgentID(c.AgentID())
	a.status.SetContext(0, status.ContextLimitFor(c.Model()))
	a.view.MarkDirty()
	a.relayout()
}

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

// Init returns the cursor blink + spinner tick so both animate from
// the first frame.
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.input.BlinkCmd(), status.SpinnerTickCmd())
}

// Update routes incoming messages.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		a.input.SetWidth(m.Width)
		a.relayout()
		return a, nil

	case events.QuitMsg:
		if a.runCancel != nil {
			a.runCancel()
		}
		return a, tea.Quit

	case events.SpinnerTickMsg:
		// Advance the spinner; re-arm the tick. Cheap enough to run
		// unconditionally — the cache layer prevents per-tick block
		// re-renders unless something actually animates.
		a.state.TickSpinner()
		// If a compaction block is animating, the transcript needs
		// to know about the new frame so its CompactingBlock bumps
		// Rev and re-renders.
		if a.transcript.HasInflightCompacting() {
			a.transcript.SetSpinnerFrame(a.state.Frame())
			a.view.MarkDirty()
		}
		return a, status.SpinnerTickCmd()

	case events.AgentEventMsg:
		return a.handleAgentEvent(m.Event)

	case events.RunDoneMsg:
		return a.handleRunDone(m.Err)

	case input.SubmitMsg:
		return a.handleSubmit(m)

	case tea.KeyMsg:
		return a.handleKey(m)
	}
	return a, nil
}

// handleAgentEvent fans one agent event through the state machine,
// the status bar, the transcript, and (on task store updates) the
// auto-fold "TASKS COMPLETE" snapshot path.
//
// The Clear that drives the auto-fold MUST run off-goroutine: each
// task deletion emits one observable.Change, which routes through
// the agent's Sink and lands as another tea.Msg. Calling Clear
// inline from Update would deadlock bubbletea v1.3.x's unbuffered
// msgs channel — the same bug v1 documented at its app.go:813-826.
// We return the Clear as a tea.Cmd so the cascade flows through
// the normal msg→Update path.
func (a *App) handleAgentEvent(e event.Event) (tea.Model, tea.Cmd) {
	a.state.Apply(e)

	if e.Usage != nil {
		a.status.SetUsage(e.Usage.Cumulative)
	}
	if a.transcript.IngestEvent(e) {
		a.view.MarkDirty()
	}
	if a.controller != nil {
		a.status.SetContext(
			a.controller.Session().LastTurnInputTokens(),
			status.ContextLimitFor(a.controller.Model()),
		)
	}

	var cmd tea.Cmd
	if e.Kind == event.KindStoreUpdate && e.StoreUpdate != nil &&
		e.StoreUpdate.Domain == task.Domain && a.controller != nil {
		if tasks.AllCompleted(a.controller.ToolState()) {
			width := a.transcriptWidth()
			snap := tasks.RenderCompleteSnapshot(a.controller.ToolState(), width, a.theme)
			a.transcript.AppendSynthetic(snap)
			a.view.MarkDirty()
			ts := a.controller.ToolState()
			cmd = func() tea.Msg {
				ts.TaskStore().Clear()
				return nil
			}
		}
	}

	// Panel content may have changed (tasks added/removed/completed,
	// subagents spawned/finished). Re-derive the viewport height so
	// new panels push the input/status up instead of off-screen.
	if e.Kind == event.KindStoreUpdate {
		a.relayout()
	}
	return a, cmd
}

// relayout recomputes the viewport height based on the current
// panel content. Called whenever a panel might have grown or
// shrunk: WindowSize, agent store updates. The transcript width
// itself doesn't depend on panel state, so we only adjust the
// vertical split.
//
// Layout vertical reservations:
//   - 5 rows: input box (3 textarea + 2 border)
//   - 2 rows: hint line + status bar
//   - ≥0 rows: tasks panel (header + N task rows)
//   - ≥0 rows: agents chip strip (one row per wrapped line)
func (a *App) relayout() {
	if a.width == 0 || a.height == 0 {
		return
	}
	used := 5 + 2 // input + hint+status
	if a.controller != nil {
		if panel := tasks.Render(a.controller.ToolState(), a.transcriptWidth(), a.theme); panel != "" {
			used += strings.Count(panel, "\n") + 1
		}
		if strip := agents.Render(a.controller.ToolState(), a.transcriptWidth(), a.theme, a.state.Frame()); strip != "" {
			used += strings.Count(strip, "\n") + 1
		}
	}
	viewportH := a.height - used
	if viewportH < 1 {
		viewportH = 1
	}
	a.view.SetSize(a.width, viewportH)
}

// transcriptWidth returns the column count panels and snapshots
// should size to. Currently identical to terminal width; future
// layout work may reserve gutter columns.
func (a *App) transcriptWidth() int {
	if a.width < 20 {
		return 20
	}
	return a.width
}

// handleRunDone fans the goroutine's exit error into the state
// machine and resets the cancel handle.
func (a *App) handleRunDone(err error) (tea.Model, tea.Cmd) {
	a.runCancel = nil
	interrupted := a.interrupted
	a.interrupted = false

	// Map the agent's interrupted error too — some providers
	// surface llm.ErrInterrupted instead of pure ctx.Cancelled.
	if errors.Is(err, llm.ErrInterrupted) {
		interrupted = true
	}
	a.state.OnRunDone(err, interrupted)
	return a, nil
}

// handleKey routes a key event. Order matters: special keys (quit,
// scroll, expand, history) precede the input textarea so multi-line
// composition with embedded special chars behaves consistently.
func (a *App) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.String() {
	case "ctrl+c":
		// Running → cancel the run; idle → quit. Matches v1.
		if a.runCancel != nil {
			a.interrupted = true
			a.runCancel()
			return a, nil
		}
		return a, tea.Quit

	case "esc":
		// Running → cancel. Error → dismiss (matches the "Esc
		// dismiss" hint). Otherwise quit.
		if a.runCancel != nil {
			a.interrupted = true
			a.runCancel()
			return a, nil
		}
		if a.state.Current() == status.StateError {
			a.state.Dismiss()
			return a, nil
		}
		return a, tea.Quit

	case "ctrl+o":
		a.transcript.ToggleExpand()
		a.view.MarkDirty()
		return a, nil

	case "pgup", "pgdown", "home", "end":
		return a, a.view.Update(m)
	}

	cmd := a.input.Update(m)
	return a, cmd
}

// handleSubmit dispatches a SubmitMsg from the Input.
//
//   - Slash commands: /exit /quit /clear handled inline; the rest
//     wait for M7's overlay focus stack.
//   - Empty submit while iter-limit-paused: Continue without
//     appending a new user message.
//   - Empty submit otherwise: no-op.
//   - Regular text: append to transcript, start (or queue) a Run.
func (a *App) handleSubmit(m input.SubmitMsg) (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.ForAgent)

	switch text {
	case "/exit", "/quit", "exit":
		a.input.Reset()
		return a, tea.Quit
	case "/clear":
		a.transcript.Reset()
		a.input.Reset()
		a.state.SetHint("")
		a.view.MarkDirty()
		return a, nil
	}

	// Iter-limit takes precedence over the empty-text check: the
	// hint tells the user "press Enter to continue", and a continue
	// takes no payload.
	if a.state.Current() == status.StateIterLimit {
		a.input.Reset()
		a.startContinue()
		return a, nil
	}

	if text == "" {
		return a, nil
	}

	if a.controller == nil {
		a.state.SetHint("no controller attached")
		return a, nil
	}

	// Mid-run submit: queue the prompt; starting a second Run
	// while one is in flight 400s on every provider.
	if a.runCancel != nil {
		a.transcript.AppendUserPrompt(m.ForView)
		a.input.Reset()
		a.controller.ToolState().UserPromptQueue().Enqueue(m.ForAgent)
		a.state.SetHint("queued — will land at next iteration")
		a.view.MarkDirty()
		return a, nil
	}

	a.transcript.AppendUserPrompt(m.ForView)
	a.input.Reset()
	a.view.MarkDirty()
	a.startRun(m.ForAgent)
	return a, nil
}

// startRun kicks off a Run in a goroutine and transitions the state
// machine to running.
func (a *App) startRun(prompt string) {
	if a.controller == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.runCancel = cancel
	a.state.OnSubmit()

	p := a.program
	go func() {
		_, err := a.controller.Run(ctx, prompt)
		if p != nil {
			p.Send(events.RunDoneMsg{Err: err})
		}
	}()
}

// startContinue resumes an iter-limit-paused run via
// controller.Continue. Same goroutine + RunDoneMsg pattern as
// startRun.
func (a *App) startContinue() {
	if a.controller == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.runCancel = cancel
	a.state.OnSubmit()

	p := a.program
	go func() {
		_, err := a.controller.Continue(ctx)
		if p != nil {
			p.Send(events.RunDoneMsg{Err: err})
		}
	}()
}

// View composes the rendered output. Vertical order (top → bottom),
// each layer collapses to zero height when its backing data is empty:
//
//	viewport / banner / transcript          (scrollable area)
//	tasks panel                             (only when tasks tracked)
//	agents chip strip                       (only when subagents tracked)
//	input box                               (rounded border)
//	hint                                    (one-liner)
//	status bar                              (HUD)
func (a *App) View() string {
	if a.width == 0 {
		return "initializing…"
	}
	var b strings.Builder
	b.WriteString(a.view.View())

	width := a.transcriptWidth()
	if a.controller != nil {
		if panel := tasks.Render(a.controller.ToolState(), width, a.theme); panel != "" {
			b.WriteByte('\n')
			b.WriteString(panel)
		}
		if strip := agents.Render(a.controller.ToolState(), width, a.theme, a.state.Frame()); strip != "" {
			b.WriteByte('\n')
			b.WriteString(strip)
		}
	}

	b.WriteByte('\n')
	b.WriteString(a.input.View())
	b.WriteByte('\n')
	// Hint line above the status bar. M7 will route focus-stack
	// providers through here; for now we pass nil and rely on
	// state-override + state-default.
	hint := status.ResolveHint(a.state, nil)
	b.WriteString(a.theme.FooterHint.Render("  " + hint))
	b.WriteByte('\n')
	b.WriteString(a.status.Compose(a.width, a.theme))
	return b.String()
}
