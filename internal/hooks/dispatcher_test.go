package hooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestDispatcher(byEvent map[Event][]Config) *Dispatcher {
	reg := NewRegistry()
	reg.ReplaceAll(byEvent)
	return NewDispatcher(reg, quietLogger(), func() BasePayload {
		return BasePayload{SessionID: "test-sess", Cwd: "/tmp"}
	}, "/tmp")
}

func TestDispatcher_FirePreToolUse_NoHooks(t *testing.T) {
	d := newTestDispatcher(nil)
	got, err := d.FirePreToolUse(context.Background(), "write", []byte(`{}`), "tool-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDispatcher_FirePreToolUse_Block(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventPreToolUse: {{
			Matcher: "write",
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo "no writes" >&2; exit 2`},
			},
		}},
	})
	got, err := d.FirePreToolUse(context.Background(), "write", []byte(`{}`), "id-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.Blocked {
		t.Fatalf("want blocked, got %+v", got)
	}
	if got.BlockReason != "no writes" {
		t.Errorf("reason=%q", got.BlockReason)
	}
}

func TestDispatcher_FirePreToolUse_PermissionDeny(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventPreToolUse: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo '{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"policy"}}'`},
			},
		}},
	})
	got, err := d.FirePreToolUse(context.Background(), "write", []byte(`{}`), "id-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.PermissionDecision != "deny" || got.Reason != "policy" {
		t.Fatalf("got=%+v", got)
	}
}

func TestDispatcher_FirePreToolUse_UpdatedInputThreads(t *testing.T) {
	// Hook 1 emits updatedInput; hook 2's input should reflect it.
	d := newTestDispatcher(map[Event][]Config{
		EventPreToolUse: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo '{"hookSpecificOutput":{"updatedInput":{"path":"/new"}}}'`},
				{Type: TypeCommand, Command: `cat`}, // echoes payload back; we just need it to succeed
			},
		}},
	})
	got, err := d.FirePreToolUse(context.Background(), "write", []byte(`{"path":"/old"}`), "id-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || string(got.UpdatedInput) != `{"path":"/new"}` {
		t.Errorf("updatedInput=%s", got.UpdatedInput)
	}
}

func TestDispatcher_FirePreToolUse_MatcherFilters(t *testing.T) {
	var hit atomic.Int32
	d := newTestDispatcher(map[Event][]Config{
		EventPreToolUse: {{
			Matcher: "write",
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo "fired" >&2; exit 2`},
			},
		}},
	})
	hit.Store(0)
	// Non-matching tool: hook should NOT fire
	got, _ := d.FirePreToolUse(context.Background(), "read", []byte(`{}`), "id-1")
	if got != nil {
		t.Errorf("read should not match write matcher; got=%+v", got)
	}
}

func TestDispatcher_FirePostToolUse_AppendsContext(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventPostToolUse: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo '{"hookSpecificOutput":{"additionalContext":"linted ok"}}'`},
			},
		}},
	})
	got, err := d.FirePostToolUse(context.Background(), "write", []byte(`{}`), "tool output", "id-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "linted ok" {
		t.Errorf("got %q", got)
	}
}

func TestDispatcher_FireUserPromptSubmit_Block(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventUserPromptSubmit: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo "filtered" >&2; exit 2`},
			},
		}},
	})
	_, blocked, reason, err := d.FireUserPromptSubmit(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !blocked || reason != "filtered" {
		t.Errorf("blocked=%v reason=%q", blocked, reason)
	}
}

func TestDispatcher_FireUserPromptSubmit_AdditionalContext(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventUserPromptSubmit: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo '{"hookSpecificOutput":{"additionalContext":"time=now"}}'`},
			},
		}},
	})
	extra, blocked, _, err := d.FireUserPromptSubmit(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("should not be blocked")
	}
	if extra != "time=now" {
		t.Errorf("extra=%q", extra)
	}
}

func TestDispatcher_FireStop_BlocksOnce(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventStop: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo "keep going" >&2; exit 2`},
			},
		}},
	})

	blocked, reason, err := d.FireStop(context.Background(), "done", false)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked || reason != "keep going" {
		t.Errorf("first pass: blocked=%v reason=%q", blocked, reason)
	}

	// Re-entry pass: stopHookActive=true must NOT block again
	blocked2, _, _ := d.FireStop(context.Background(), "done", true)
	if blocked2 {
		t.Errorf("second pass with stop_hook_active should not block")
	}
}

func TestDispatcher_FireSessionStart_InitialUserMessage(t *testing.T) {
	d := newTestDispatcher(map[Event][]Config{
		EventSessionStart: {{
			Hooks: []Command{
				{Type: TypeCommand, Command: `echo '{"hookSpecificOutput":{"initialUserMessage":"context: " ,"additionalContext":"ctx"}}'`},
			},
		}},
	})
	init, extra, err := d.FireSessionStart(context.Background(), "startup", "claude-opus")
	if err != nil {
		t.Fatal(err)
	}
	if init != "context: " || extra != "ctx" {
		t.Errorf("init=%q extra=%q", init, extra)
	}
}

func TestDispatcher_FireNotification_HTTP(t *testing.T) {
	var hits atomic.Int32
	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p NotificationPayload
		_ = json.NewDecoder(r.Body).Decode(&p)
		if p.NType == "approval_needed" {
			hits.Add(1)
		}
		w.WriteHeader(200)
		select {
		case done <- struct{}{}:
		default:
		}
	}))
	defer srv.Close()

	// async=false so we can synchronously verify the HTTP hit
	d := newTestDispatcher(map[Event][]Config{
		EventNotification: {{
			Hooks: []Command{
				{Type: TypeHTTP, URL: srv.URL, Async: false},
			},
		}},
	})
	d.FireNotification(context.Background(), "review needed", "evva", "approval_needed")
	if hits.Load() != 1 {
		t.Errorf("hits=%d", hits.Load())
	}
}

func TestDispatcher_NilSafe(t *testing.T) {
	var d *Dispatcher
	if d.Has(EventPreToolUse) {
		t.Error("nil Has should be false")
	}
	got, err := d.FirePreToolUse(context.Background(), "x", nil, "")
	if err != nil || got != nil {
		t.Error("nil dispatcher should noop")
	}
	d.FireNotification(context.Background(), "m", "t", "n")
}
