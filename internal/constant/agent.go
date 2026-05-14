package constant

type AgentStatus string

const (
	INIT       AgentStatus = "init"
	THINKING   AgentStatus = "thinking"
	TOOL_USING AgentStatus = "tool_using"
	IDLE       AgentStatus = "idle"
	SAVING     AgentStatus = "saving"
	COMPACTING AgentStatus = "compacting"
	SHUTDOWN   AgentStatus = "shutdown"
)
