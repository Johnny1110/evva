package session

import (
	"path/filepath"

	"github.com/johnny1110/evva/internal/llm"
)

// Session holds the live conversation history for a single agent run.
// The agent appends every message (user, assistant, tool result) here so the
// LLM always receives the full context on the next turn.
// tools, agent, llm, tui will use it.
type Session struct {
	// LLM context payload
	Messages []llm.Message
	// microCompacted: compress tool_use result block only (level-1 compact)
	microCompacted bool
	// fullCompact: compress all session message (level-2 compact)
	fullCompactCount int
}

// ReadTracker records file paths the agent has called read_file on.
// Zero value is ready to use.
type ReadTracker struct {
	seen map[string]struct{}
}

// MarkRead records that path was read.
func (t *ReadTracker) MarkRead(absPath string) {
	if t.seen == nil {
		t.seen = make(map[string]struct{})
	}
	t.seen[filepath.Clean(absPath)] = struct{}{}
}

// WasRead reports whether path has been marked via MarkRead.
func (t *ReadTracker) WasRead(absPath string) bool {
	if t.seen == nil {
		return false
	}
	_, ok := t.seen[filepath.Clean(absPath)]
	return ok
}

func New() *Session {
	return &Session{}
}

func (s *Session) Append(msg llm.Message) {
	s.Messages = append(s.Messages, msg)
}

func (s *Session) GetMessages() []llm.Message {
	return s.Messages
}

func (s *Session) IsMicroCompacted() bool {
	return s.microCompacted
}

func (s *Session) MicroCompact(messages []llm.Message) {
	s.microCompacted = true
	s.Messages = messages
}

func (s *Session) FullCompact(messages []llm.Message) {
	s.microCompacted = false
	s.fullCompactCount++
	s.Messages = messages
}

func (s *Session) GetFullCompactCount() int {
	return s.fullCompactCount
}
