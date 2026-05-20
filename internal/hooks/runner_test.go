package hooks

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunCommand_HappyPath(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `cat > /dev/null; echo '{"continue":true}'`}
	res := runCommand(context.Background(), quietLogger(), cmd, []byte(`{"x":1}`), nil, 5*time.Second)
	if res.exitCode != 0 || res.timedOut || res.err != nil {
		t.Fatalf("res=%+v", res)
	}
	if !strings.Contains(string(res.stdout), "continue") {
		t.Errorf("stdout=%s", res.stdout)
	}
}

func TestRunCommand_Exit2(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `echo "nope" >&2; exit 2`}
	res := runCommand(context.Background(), quietLogger(), cmd, nil, nil, 5*time.Second)
	if res.exitCode != 2 {
		t.Fatalf("exit=%d want 2", res.exitCode)
	}
	if reason := extractReason(res, ""); reason != "nope" {
		t.Errorf("reason=%q", reason)
	}
}

func TestRunCommand_Exit1NonBlocking(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `exit 1`}
	res := runCommand(context.Background(), quietLogger(), cmd, nil, nil, 5*time.Second)
	if res.exitCode != 1 {
		t.Fatalf("exit=%d want 1", res.exitCode)
	}
}

func TestRunCommand_Timeout(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `sleep 2`, Timeout: 1}
	res := runCommand(context.Background(), quietLogger(), cmd, nil, nil, 5*time.Second)
	if !res.timedOut {
		t.Fatalf("expected timeout, got %+v", res)
	}
}

func TestRunCommand_StdinPayload(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `cat`}
	res := runCommand(context.Background(), quietLogger(), cmd, []byte(`payload-marker`), nil, 5*time.Second)
	if res.exitCode != 0 {
		t.Fatalf("exit=%d", res.exitCode)
	}
	if string(res.stdout) != "payload-marker" {
		t.Errorf("stdout=%q", res.stdout)
	}
}

func TestRunCommand_EnvForwarded(t *testing.T) {
	cmd := Command{Type: TypeCommand, Command: `echo "$EVVA_AGENT_ID"`}
	env := []string{"EVVA_AGENT_ID=abc123"}
	res := runCommand(context.Background(), quietLogger(), cmd, nil, env, 5*time.Second)
	if strings.TrimSpace(string(res.stdout)) != "abc123" {
		t.Errorf("stdout=%q", res.stdout)
	}
}

func TestRunCommand_Empty(t *testing.T) {
	res := runCommand(context.Background(), quietLogger(), Command{Type: TypeCommand}, nil, nil, time.Second)
	if res.err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestExtractReason_FallbackOrder(t *testing.T) {
	if got := extractReason(commandResult{stderr: []byte("err"), stdout: []byte("out")}, "fb"); got != "err" {
		t.Errorf("stderr should win: %q", got)
	}
	if got := extractReason(commandResult{stdout: []byte("out")}, "fb"); got != "out" {
		t.Errorf("stdout fallback: %q", got)
	}
	if got := extractReason(commandResult{}, "fb"); got != "fb" {
		t.Errorf("fallback: %q", got)
	}
}
