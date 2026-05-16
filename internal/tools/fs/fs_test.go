package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "github.com/johnny1110/evva/configs"
)

// TestResolvePathExpandsTilde locks down that `~/x` expands using the
// invoking user's home directory. Reported bug: under sudo (or any env
// where $HOME=/root but the user is non-root), `~/tmp` resolved to
// `/root/tmp` instead of the user's actual home. With SUDO_USER honored
// in resolveUserHome the expansion now follows the user's intent.
func TestResolvePathExpandsTilde(t *testing.T) {
	t.Setenv("SUDO_USER", "")
	t.Setenv("HOME", "/home/agent")

	got, err := resolvePath("~/tmp/notes.md")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	want := "/home/agent/tmp/notes.md"
	if got != want {
		t.Fatalf("resolvePath(~/tmp/notes.md) = %q, want %q", got, want)
	}
}

// TestResolvePathHonorsSudoUser proves that when $HOME points at /root
// (running under sudo) but $SUDO_USER names the real user, `~/x`
// resolves against the real user's home — the exact bug the user filed.
//
// We use user.Lookup so this test only runs meaningfully when the test
// process has a valid passwd entry available; skip otherwise.
func TestResolvePathHonorsSudoUser(t *testing.T) {
	// Skip on systems where we can't look up the current user (rare).
	username := lookupCurrentUsername(t)

	t.Setenv("HOME", "/root")
	t.Setenv("SUDO_USER", username)

	got, err := resolvePath("~/tmp/notes.md")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	if strings.HasPrefix(got, "/root/") {
		t.Fatalf("path leaked /root — got %q (HOME shadowed SUDO_USER)", got)
	}
	if !strings.HasSuffix(got, "/tmp/notes.md") {
		t.Fatalf("unexpected expansion: %q", got)
	}
}

// TestResolvePathAutoAbs locks down the second half of the fix:
// relative paths get auto-promoted to absolute via cfg.WorkDir
// instead of being rejected. Lets the model write "notes/todo.md"
// without plumbing the workdir itself.
func TestResolvePathAutoAbs(t *testing.T) {
	wd := t.TempDir()
	cfg := config.Get()
	prev := cfg.WorkDir
	cfg.WorkDir = wd
	t.Cleanup(func() { cfg.WorkDir = prev })

	got, err := resolvePath("notes/todo.md")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	want := filepath.Join(wd, "notes", "todo.md")
	if got != want {
		t.Fatalf("resolvePath(notes/todo.md) = %q, want %q", got, want)
	}
}

// TestResolvePathPreservesAbsolute confirms an already-absolute path
// passes through untouched (modulo filepath.Clean).
func TestResolvePathPreservesAbsolute(t *testing.T) {
	got, err := resolvePath("/var/log/agent.log")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	if got != "/var/log/agent.log" {
		t.Fatalf("resolvePath(/var/log/agent.log) = %q", got)
	}
}

// TestResolvePathRejectsEmpty keeps the empty-input contract — every
// fs tool relies on this guard to surface "file_path is required"
// rather than silently operating on the workdir root.
func TestResolvePathRejectsEmpty(t *testing.T) {
	if _, err := resolvePath(""); err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func lookupCurrentUsername(t *testing.T) string {
	t.Helper()
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	if u := strings.TrimSpace(os.Getenv("LOGNAME")); u != "" {
		return u
	}
	t.Skip("cannot determine current username from env")
	return ""
}
