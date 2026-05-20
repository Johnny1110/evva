package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/johnny1110/evva/internal/agent/sysprompt"
)

// zeroPromptContext returns a zero-value PromptContext. Disk-agent
// BuildSystemPrompt closures capture the body and ignore the context, so
// the value doesn't matter — we just need any PromptContext to call them.
func zeroPromptContext() sysprompt.PromptContext { return sysprompt.PromptContext{} }

// writeAgent writes a complete agent directory: system_prompt.md, tools.yml,
// and meta.yml. Used by every test that needs a valid baseline.
func writeAgent(t *testing.T, root, name, prompt, toolsBody, metaBody string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	must := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if prompt != "" {
		must(filepath.Join(dir, "system_prompt.md"), prompt)
	}
	if toolsBody != "" {
		must(filepath.Join(dir, "tools.yml"), toolsBody)
	}
	if metaBody != "" {
		must(filepath.Join(dir, "meta.yml"), metaBody)
	}
}

func TestLoad_EmptyHomeReturnsNothing(t *testing.T) {
	defs, warns := Load("")
	if len(defs) != 0 || len(warns) != 0 {
		t.Fatalf("expected no results for empty home, got defs=%v warns=%v", defs, warns)
	}
}

func TestLoad_MissingAgentsDirIsSilent(t *testing.T) {
	home := t.TempDir() // no agents/ subdir
	defs, warns := Load(home)
	if len(defs) != 0 || len(warns) != 0 {
		t.Fatalf("expected silent skip when agents/ missing, got %v / %v", defs, warns)
	}
}

func TestLoad_ValidSubagent(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "test-agent",
		"You are a test agent.\n",
		"active: [read_file, bash]\ndeferred: [task_create]\n",
		"as: [subagent]\nmodel: \"\"\nwhen_to_use: A test agent for unit tests.\n",
	)
	defs, warns := Load(home)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	d := defs[0]
	if d.Name != "test-agent" {
		t.Errorf("name: got %q", d.Name)
	}
	if !d.IsSubagent() || d.IsMain() {
		t.Errorf("visibility: want subagent-only, got As=%v", d.As)
	}
	if d.WhenToUse != "A test agent for unit tests." {
		t.Errorf("when_to_use: got %q", d.WhenToUse)
	}
	if len(d.ActiveTools) != 2 || string(d.ActiveTools[0]) != "read_file" {
		t.Errorf("active_tools: got %v", d.ActiveTools)
	}
	if len(d.DeferredTools) != 1 || string(d.DeferredTools[0]) != "task_create" {
		t.Errorf("deferred_tools: got %v", d.DeferredTools)
	}
	// BuildSystemPrompt is exercised in the dedicated test below.
}

func TestLoad_BuildSystemPromptReturnsDiskBody(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	body := "You are a focused agent.\nFollow these rules:\n- Rule 1\n- Rule 2\n"
	writeAgent(t, agentsDir, "focused", body,
		"active: []\ndeferred: []\n",
		"as: [subagent]\n",
	)
	defs, _ := Load(home)
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	// The closure ignores its PromptContext, so we can pass the zero value.
	got := defs[0].BuildSystemPrompt(zeroPromptContext())
	if got != body {
		t.Errorf("BuildSystemPrompt: got %q, want %q", got, body)
	}
}

func TestLoad_MissingSystemPromptIsWarning(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "broken", "",
		"active: []\ndeferred: []\n",
		"as: [subagent]\n",
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Fatalf("expected 0 defs for broken agent, got %d", len(defs))
	}
	if len(warns) == 0 {
		t.Fatal("expected a warning for missing system_prompt.md")
	}
	if warns[0].Agent != "broken" {
		t.Errorf("warning.Agent: got %q", warns[0].Agent)
	}
}

func TestLoad_EmptySystemPromptIsWarning(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "empty", "   \n  \n",
		"active: []\n",
		"as: [subagent]\n",
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for empty system_prompt.md")
	}
}

func TestLoad_MalformedToolsYmlIsWarning(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "bad-yaml", "body\n",
		"active: [unclosed,\n", // malformed YAML
		"as: [subagent]\n",
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for malformed tools.yml")
	}
}

func TestLoad_MissingAsFieldIsWarning(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "missing-as", "body\n",
		"active: []\n",
		"when_to_use: nothing\n",
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for missing as field")
	}
}

func TestLoad_InvalidAsValueIsWarning(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "weird-as", "body\n",
		"active: []\n",
		"as: [main, intern]\n", // "intern" is not a valid value
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for invalid as value")
	}
}

func TestLoad_MainAndSubagent(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "hybrid", "body\n",
		"active: [bash]\n",
		"as: [main, subagent]\n",
	)
	defs, warns := Load(home)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	d := defs[0]
	if !d.IsMain() || !d.IsSubagent() {
		t.Errorf("expected hybrid (main+subagent), got As=%v", d.As)
	}
}

// TestLoad_MemoryAndSkillsFlagsDefaultOff verifies the legacy contract
// holds when meta.yml omits inject_memory / advertise_skills: disk
// personas stay minimal unless they opt in.
func TestLoad_MemoryAndSkillsFlagsDefaultOff(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "lite", "body\n",
		"active: [bash]\n",
		"as: [main]\n", // no inject_memory / advertise_skills keys
	)
	defs, warns := Load(home)
	if len(warns) != 0 || len(defs) != 1 {
		t.Fatalf("warns=%v defs=%d", warns, len(defs))
	}
	d := defs[0]
	if !d.OmitMemory {
		t.Errorf("default should OmitMemory=true (inject_memory unset)")
	}
	if d.AdvertiseSkills {
		t.Errorf("default should AdvertiseSkills=false (advertise_skills unset)")
	}
}

// TestLoad_MemoryAndSkillsFlagsRespected covers the opt-in path: a
// main-tier disk persona that wants memory + skills flips the YAML
// flags and gets them.
func TestLoad_MemoryAndSkillsFlagsRespected(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, "rich", "body\n",
		"active: [bash]\n",
		"as: [main]\ninject_memory: true\nadvertise_skills: true\n",
	)
	defs, warns := Load(home)
	if len(warns) != 0 || len(defs) != 1 {
		t.Fatalf("warns=%v defs=%d", warns, len(defs))
	}
	d := defs[0]
	if d.OmitMemory {
		t.Errorf("inject_memory: true should set OmitMemory=false")
	}
	if !d.AdvertiseSkills {
		t.Errorf("advertise_skills: true should be reflected")
	}
}

func TestLoad_SkipsHiddenDirectories(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, "agents")
	writeAgent(t, agentsDir, ".hidden", "body\n",
		"active: []\n",
		"as: [subagent]\n",
	)
	defs, warns := Load(home)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs (hidden dir skipped), got %d", len(defs))
	}
	if len(warns) != 0 {
		t.Errorf("expected 0 warnings, got %v", warns)
	}
}
