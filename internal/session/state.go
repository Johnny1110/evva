package session

import "github.com/johnny1110/evva/internal/session/task"

type State struct {
	TaskStore task.Store
}

// NewState for new agent
func NewState() *State {
	return &State{
		TaskStore: task.NewStore(),
	}
}
