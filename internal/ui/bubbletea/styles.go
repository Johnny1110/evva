package bubbletea

import "github.com/charmbracelet/lipgloss"

// Miku palette — 24-bit truecolor. Every lipgloss color in the TUI maps
// to one of these constants so the theme has a single source of truth.
// Exported names are deliberate: future widgets / external themes can
// reference the same tokens.
const (
	paletteBg      lipgloss.Color = "#121218"
	paletteFg      lipgloss.Color = "#E8F6F5"
	paletteCursor  lipgloss.Color = "#39C5BB"
	paletteDim     lipgloss.Color = "#1E1E24"
	paletteRed     lipgloss.Color = "#FF5FA2"
	paletteGreen   lipgloss.Color = "#39C5BB"
	paletteYellow  lipgloss.Color = "#FFD166"
	paletteBlue    lipgloss.Color = "#4D8BFF"
	paletteMagenta lipgloss.Color = "#C792EA"
	paletteCyan    lipgloss.Color = "#7FE7DE"
	paletteWhite   lipgloss.Color = "#E8F6F5"
	// paletteOrange is the executing-spinner color — the user asked
	// for a distinct orange tone separate from the yellow used by
	// compaction / iter-limit so "model is running tools" reads
	// differently from "model is paused / housekeeping".
	paletteOrange lipgloss.Color = "#FF9F40"
	// paletteLightBlue is the thinking-spinner color — sky-cool so it
	// reads as "model is reasoning" without competing visually with
	// the prompt-blue used for the user line and panel headers.
	paletteLightBlue lipgloss.Color = "#8FB7FF"
	// paletteMuted is a softer grey for status-bar keys and the
	// footer hint — distinguishable from foreground without going as
	// dim as paletteDim.
	paletteMuted lipgloss.Color = "#6B6F7A"
)

// styles is the central palette. Each style is built once at package
// init time. Truecolor is unconditional — evva targets modern terminals.
var styles = struct {
	UserPrompt  lipgloss.Style
	Assistant   lipgloss.Style
	Thinking    lipgloss.Style
	ToolCall    lipgloss.Style
	ToolOK      lipgloss.Style
	ToolErr     lipgloss.Style
	ToolResult  lipgloss.Style
	DiffAdd     lipgloss.Style
	DiffRemove  lipgloss.Style
	DiffContext lipgloss.Style
	DiffHeader  lipgloss.Style
	PanelHeader lipgloss.Style
	PanelRow    lipgloss.Style
	PanelBorder lipgloss.Style
	StatusBar   lipgloss.Style
	StatusKey   lipgloss.Style
	StatusValue lipgloss.Style
	ErrorBanner lipgloss.Style
	InputBorder lipgloss.Style
	DimText     lipgloss.Style
	Banner      lipgloss.Style
	BannerBox   lipgloss.Style
	Greeting    lipgloss.Style
	Compacting  lipgloss.Style
	Draining    lipgloss.Style
	FooterHint  lipgloss.Style
	TasksDone   lipgloss.Style
	PasteChip   lipgloss.Style
	Timeline    lipgloss.Style
	TimelineCut lipgloss.Style
	ContextBar  lipgloss.Style
	ContextFill lipgloss.Style
	ContextRail lipgloss.Style
	SpinThink   lipgloss.Style
	SpinExec    lipgloss.Style
}{
	UserPrompt:  lipgloss.NewStyle().Foreground(paletteBlue).Bold(true),
	Assistant:   lipgloss.NewStyle().Foreground(paletteWhite),
	Thinking:    lipgloss.NewStyle().Foreground(paletteMuted).Italic(true),
	ToolCall:    lipgloss.NewStyle().Foreground(paletteOrange).Bold(true),
	ToolOK:      lipgloss.NewStyle().Foreground(paletteGreen),
	ToolErr:     lipgloss.NewStyle().Foreground(paletteRed),
	ToolResult:  lipgloss.NewStyle().Foreground(paletteCyan),
	DiffAdd:     lipgloss.NewStyle().Foreground(paletteGreen),
	DiffRemove:  lipgloss.NewStyle().Foreground(paletteRed),
	DiffContext: lipgloss.NewStyle().Foreground(paletteMuted),
	DiffHeader:  lipgloss.NewStyle().Foreground(paletteMuted).Italic(true),
	PanelHeader: lipgloss.NewStyle().Foreground(paletteBlue).Bold(true),
	PanelRow:    lipgloss.NewStyle().Foreground(paletteFg),
	PanelBorder: lipgloss.NewStyle().Foreground(paletteDim),
	StatusBar:   lipgloss.NewStyle().Foreground(paletteFg).Background(paletteDim).Padding(0, 1),
	StatusKey:   lipgloss.NewStyle().Foreground(paletteMuted),
	StatusValue: lipgloss.NewStyle().Foreground(paletteFg).Bold(true),
	ErrorBanner: lipgloss.NewStyle().Foreground(paletteRed).Bold(true),
	InputBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(paletteCyan).Padding(0, 1),
	DimText:     lipgloss.NewStyle().Foreground(paletteMuted),
	Banner: lipgloss.NewStyle().Foreground(paletteMagenta),
	BannerBox: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(paletteCyan).
		Padding(1, 2),
	Greeting:   lipgloss.NewStyle().Foreground(paletteCyan).Italic(true),
	Compacting: lipgloss.NewStyle().Foreground(paletteYellow).Italic(true),
	Draining:   lipgloss.NewStyle().Foreground(paletteBlue).Italic(true),
	FooterHint: lipgloss.NewStyle().Foreground(paletteMuted),
	TasksDone:   lipgloss.NewStyle().Foreground(paletteGreen).Bold(true),
	PasteChip:   lipgloss.NewStyle().Foreground(paletteMagenta).Italic(true),
	Timeline:    lipgloss.NewStyle().Foreground(paletteMuted),
	TimelineCut: lipgloss.NewStyle().Foreground(paletteBlue),
	ContextBar:  lipgloss.NewStyle().Foreground(paletteMuted),
	ContextFill: lipgloss.NewStyle().Foreground(paletteCyan),
	ContextRail: lipgloss.NewStyle().Foreground(paletteDim),
	SpinThink:   lipgloss.NewStyle().Foreground(paletteLightBlue).Bold(true),
	SpinExec:    lipgloss.NewStyle().Foreground(paletteOrange).Bold(true),
}
