package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	reg, warns := Load(dir, dir)
	if len(warns) != 0 {
		t.Errorf("missing files: warns=%v", warns)
	}
	if reg.HasAny(EventPreToolUse) {
		t.Error("expected empty registry")
	}
}

func TestLoad_ProjectAndUserMerge(t *testing.T) {
	workdir := t.TempDir()
	evvaHome := t.TempDir()

	writeJSON(t, filepath.Join(workdir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks":   []any{map[string]any{"type": "command", "command": "echo project"}},
				},
			},
		},
	})
	writeJSON(t, filepath.Join(evvaHome, "settings.json"), map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks":   []any{map[string]any{"type": "command", "command": "echo user"}},
				},
			},
		},
	})

	reg, warns := Load(workdir, evvaHome)
	if len(warns) != 0 {
		t.Errorf("unexpected warns: %v", warns)
	}
	cfgs := reg.For(EventPreToolUse)
	if len(cfgs) != 2 {
		t.Fatalf("want 2 configs, got %d", len(cfgs))
	}
	if cfgs[0].Hooks[0].Command != "echo project" {
		t.Errorf("project should be first; got %q", cfgs[0].Hooks[0].Command)
	}
	if cfgs[1].Hooks[0].Command != "echo user" {
		t.Errorf("user should be second; got %q", cfgs[1].Hooks[0].Command)
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".evva", "settings.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte("{not json"), 0o644)

	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for malformed JSON")
	}
}

func TestLoad_UnknownEvent(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"NotARealEvent": []any{
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "x"}}},
			},
		},
	})
	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for unknown event")
	}
}

func TestLoad_BadHTTPURL(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"Notification": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "http", "url": "not a url"},
					},
				},
			},
		},
	})
	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for bad URL")
	}
}

func TestLoad_MissingCommand(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command"},
					},
				},
			},
		},
	})
	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for missing command")
	}
}

func TestLoad_UnknownType(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "lol", "command": "x"},
					},
				},
			},
		},
	})
	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for unknown type")
	}
}

func TestLoad_TimeoutRange(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": "x", "timeout": 9999},
					},
				},
			},
		},
	})
	_, warns := Load(dir, "")
	if len(warns) == 0 {
		t.Fatal("expected warning for out-of-range timeout")
	}
}

func TestLoad_HTTPDefaultsAsync(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".evva", "settings.json"), map[string]any{
		"hooks": map[string]any{
			"Notification": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "http", "url": "https://example.com/hook"},
					},
				},
			},
		},
	})
	reg, warns := Load(dir, "")
	if len(warns) != 0 {
		t.Fatalf("unexpected warns: %v", warns)
	}
	cfgs := reg.For(EventNotification)
	if !cfgs[0].Hooks[0].Async {
		t.Error("http hook should default to async=true")
	}
}
