// Package fs exposes filesystem tools (Read, Write, Edit) as stateless
// singletons. Construction policy (eager vs lazy) is decided by the agent;
// this package only knows how to produce tool instances.
package fs

import (
	"fmt"
	"os"
	"path/filepath"

	config "github.com/johnny1110/evva/configs"
	"github.com/johnny1110/evva/internal/tools"
)

// Names lists every tool name this package contributes, in canonical order.
func Names() []tools.ToolName {
	return []tools.ToolName{tools.READ_FILE, tools.WRITE_FILE, tools.EDIT_FILE}
}

// resolvePath validates that pathStr is absolute and returns its cleaned
// form. Matches Claude Code's contract — the LLM must supply absolute
// paths; relative paths are rejected up front with a hint pointing at the
// workdir, so a misconfigured agent never silently writes to /cwd by
// mistake.
//
// A leading `~` or `~/` is expanded to the invoking user's home
// directory before the absolute-path check. Models commonly write
// `~/tmp/...` and a strict rejection there is unhelpful when expansion
// is unambiguous.
func resolvePath(pathStr string) (string, error) {
	if pathStr == "" {
		return "", fmt.Errorf("file_path is required")
	}
	expanded, err := expandHome(pathStr)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		cfg := config.Get()
		return "", fmt.Errorf("file_path must be absolute (relative paths are not supported; workdir is %s)", cfg.WorkDir)
	}
	return filepath.Clean(expanded), nil
}

// expandHome resolves a leading `~` or `~/` against the current user's
// home directory. Any other tilde (e.g. `~bob/foo`) is left untouched —
// per-user lookup is out of scope for now.
func expandHome(p string) (string, error) {
	if p == "" || p[0] != '~' {
		return p, nil
	}
	if len(p) > 1 && p[1] != '/' && p[1] != filepath.Separator {
		// `~bob/...` style — unsupported; pass through so the
		// absolute-path check reports a clear error.
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand ~: %w", err)
	}
	if len(p) == 1 {
		return home, nil
	}
	return filepath.Join(home, p[2:]), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
