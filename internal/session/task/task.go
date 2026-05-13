package task

// Status enumerates the lifecycle states a task can be in.
type STATUS string

const (
	STATUS_PENDING    STATUS = "pending"
	STATUS_INPROGRESS STATUS = "in_progress"
	STATUS_COMPLETED  STATUS = "completed"
	STATUS_DELETED    STATUS = "deleted"
)

// Task is the in-memory record the task tools operate on.
type Task struct {
	ID          string
	Title       string
	Description string
	Status      STATUS
}

// Store is the per-agent backing store for the task tools. All six task tools
// (Create, Get, List, Update, Output, Stop) share one Store via constructor
// injection, so they cooperate without any global state.
//
// Safe for concurrent access — the agent loop and TUI may read simultaneously.
// The fields are still unexported; once the tool methods land, the public
// surface will be the Store's own methods (Add/Update/etc).
type Store map[string]*Task

func NewStore() Store {
	return make(map[string]*Task)
}
