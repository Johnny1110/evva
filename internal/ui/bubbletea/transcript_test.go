package bubbletea

import (
	"strings"
	"testing"
)

// TestWrapForWidthPreservesIndent locks down the fix for the
// "pasted code loses leading spaces after wrap" regression:
// muesli/reflow's wrap drops whitespace after a forced line break by
// default, which makes indented code paste look truncated to the user.
// wrapForWidth has to opt into PreserveSpace so indentation survives.
func TestWrapForWidthPreservesIndent(t *testing.T) {
	// 60-col line of indented code that needs a forced break at 40.
	input := "    " + strings.Repeat("abcdefghij", 6)
	out := wrapForWidth(input, 40)

	// Every input rune must appear in the output (modulo any newlines
	// the wrapper introduces). The leading 4 spaces are the smoking
	// gun — they must survive the wrap.
	inRunes := []rune(strings.ReplaceAll(input, "\n", ""))
	outRunes := []rune(strings.ReplaceAll(out, "\n", ""))
	if string(inRunes) != string(outRunes) {
		t.Fatalf("content not preserved through wrap\n want=%q\n  got=%q", input, out)
	}
	if !strings.HasPrefix(out, "    ") {
		t.Fatalf("leading indent dropped\n got=%q", out)
	}
}

// TestWrapForWidthPreservesNewlines verifies a multi-line paste keeps
// every original newline through both wrap passes.
func TestWrapForWidthPreservesNewlines(t *testing.T) {
	input := "line one\nline two\nline three"
	out := wrapForWidth(input, 80)
	if !strings.Contains(out, "line one") ||
		!strings.Contains(out, "line two") ||
		!strings.Contains(out, "line three") {
		t.Fatalf("paste lines lost\n in=%q\nout=%q", input, out)
	}
	if strings.Count(out, "\n") < 2 {
		t.Fatalf("newlines collapsed: %q", out)
	}
}

// TestRenderUserPromptPreservesPaste runs the full transcript path
// — user prompt with embedded paste content + chips — and checks no
// runes are lost. This is the regression the user reported: paste
// content getting "cut" in the conversation history.
func TestRenderUserPromptPreservesPaste(t *testing.T) {
	tr := transcript{width: 60, textInflightIdx: -1, thinkingInflightIdx: -1, bannerIdx: -1}

	paste := "func main() {\n    for i := 0; i < 100; i++ {\n        doSomethingExpensive(i)\n    }\n}"
	body := "here's the code I'm worried about:\n" +
		styles.PasteChip.Render("╔═ PASTE 80 chars ═╗") + "\n" +
		paste + "\n" +
		styles.PasteChip.Render("╚════════════════════╝") + "\n"
	tr.appendUserPrompt(body)

	out := tr.String()
	// Every line of the paste must still be present in the rendered
	// transcript. We check substrings (not whole-string equality)
	// because the renderer adds the scanline header + styling.
	for _, line := range strings.Split(paste, "\n") {
		if line == "" {
			continue
		}
		if !strings.Contains(out, strings.TrimLeft(line, " ")) {
			t.Fatalf("paste line missing from transcript\n line=%q\n  out=\n%s", line, out)
		}
	}
	// The full first indent token must survive even after wrap forces
	// breaks on long lines.
	if !strings.Contains(out, "    for i :=") {
		t.Fatalf("indented code lost leading spaces\n out=\n%s", out)
	}
}
