package hooks

import (
	"testing"
)

func TestParseDecision_Empty(t *testing.T) {
	if got := parseDecision(nil); got.Decision != "" || got.Continue != nil {
		t.Fatalf("nil stdout: want empty, got %+v", got)
	}
	if got := parseDecision([]byte("   \n  ")); got.Decision != "" {
		t.Fatalf("whitespace: want empty, got %+v", got)
	}
}

func TestParseDecision_NotJSON(t *testing.T) {
	if got := parseDecision([]byte("hello world")); got.Decision != "" {
		t.Fatalf("plain text: want empty, got %+v", got)
	}
}

func TestParseDecision_Fields(t *testing.T) {
	in := []byte(`{
		"continue": false,
		"decision": "block",
		"reason": "no good",
		"systemMessage": "warn",
		"hookSpecificOutput": {
			"permissionDecision": "deny",
			"permissionDecisionReason": "compliance",
			"additionalContext": "ctx",
			"initialUserMessage": "hi"
		}
	}`)
	got := parseDecision(in)
	if got.Continue == nil || *got.Continue != false {
		t.Errorf("continue: %v", got.Continue)
	}
	if got.Decision != "block" || got.Reason != "no good" || got.SystemMessage != "warn" {
		t.Errorf("top-level: %+v", got)
	}
	if extractAdditionalContext(got) != "ctx" {
		t.Errorf("additionalContext")
	}
	if extractInitialUserMessage(got) != "hi" {
		t.Errorf("initialUserMessage")
	}
}

func TestApplyPreToolUse_Block(t *testing.T) {
	cf := false
	acc := &PreToolUseDecision{}
	if !applyPreToolUse(acc, Decision{Continue: &cf, Reason: "stop"}) {
		t.Fatal("continue=false should stop")
	}
	if !acc.Blocked || acc.BlockReason != "stop" {
		t.Errorf("acc=%+v", acc)
	}

	acc2 := &PreToolUseDecision{}
	if !applyPreToolUse(acc2, Decision{Decision: "block", Reason: "no"}) {
		t.Fatal("decision=block should stop")
	}
	if !acc2.Blocked {
		t.Errorf("acc2=%+v", acc2)
	}
}

func TestApplyPreToolUse_Approve(t *testing.T) {
	acc := &PreToolUseDecision{}
	if applyPreToolUse(acc, Decision{Decision: "approve", Reason: "yes"}) {
		t.Fatal("approve shouldn't stop")
	}
	if acc.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision=%q", acc.PermissionDecision)
	}
}

func TestApplyPreToolUse_PermissionDecisionOverride(t *testing.T) {
	acc := &PreToolUseDecision{}
	d := Decision{
		HookSpecificOutput: map[string]any{
			"permissionDecision":       "deny",
			"permissionDecisionReason": "policy",
		},
	}
	if applyPreToolUse(acc, d) {
		t.Fatal("not blocking")
	}
	if acc.PermissionDecision != "deny" || acc.Reason != "policy" {
		t.Errorf("acc=%+v", acc)
	}
}

func TestApplyPreToolUse_UpdatedInput(t *testing.T) {
	acc := &PreToolUseDecision{}
	d := Decision{
		HookSpecificOutput: map[string]any{
			"updatedInput": map[string]any{"path": "/new"},
		},
	}
	applyPreToolUse(acc, d)
	if len(acc.UpdatedInput) == 0 {
		t.Fatal("expected updatedInput marshalled")
	}
	s := string(acc.UpdatedInput)
	if s != `{"path":"/new"}` {
		t.Errorf("got %s", s)
	}
}

func TestApplyPreToolUse_AdditionalContextAccumulates(t *testing.T) {
	acc := &PreToolUseDecision{}
	applyPreToolUse(acc, Decision{HookSpecificOutput: map[string]any{"additionalContext": "first"}})
	applyPreToolUse(acc, Decision{HookSpecificOutput: map[string]any{"additionalContext": "second"}})
	if acc.AdditionalContext != "first\nsecond" {
		t.Errorf("got %q", acc.AdditionalContext)
	}
}

func TestIsBlock(t *testing.T) {
	cf := false
	if b, _ := isBlock(Decision{Continue: &cf}); !b {
		t.Error("continue=false")
	}
	if b, _ := isBlock(Decision{Decision: "BLOCK"}); !b {
		t.Error("case-insensitive block")
	}
	if b, _ := isBlock(Decision{Decision: "approve"}); b {
		t.Error("approve not block")
	}
}
