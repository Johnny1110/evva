package task

import "github.com/johnny1110/evva/internal/tools"

// taskNames is the canonical order for the task subsystem. The Group's build
// must return instances in the same order — build resolves a tool by its
// member index, so order is load-bearing.
var taskNames = []tools.ToolName{
	tools.TASK_CREATE,
	tools.TASK_GET,
	tools.TASK_LIST,
	tools.TASK_UPDATE,
	tools.TASK_OUTPUT,
	tools.TASK_STOP,
}

var CREATE tools.Tool = &CreateTool{}
var GET tools.Tool = &GetTool{}
var LIST tools.Tool = &ListTool{}
var UPDATE tools.Tool = &UpdateTool{}
var OUTPUT tools.Tool = &OutputTool{}
var STOP tools.Tool = &StopTool{}

func init() {
	tools.Register(tools.TASK_CREATE, CREATE)
	tools.Register(tools.TASK_GET, GET)
	tools.Register(tools.TASK_LIST, LIST)
	tools.Register(tools.TASK_UPDATE, UPDATE)
	tools.Register(tools.TASK_OUTPUT, OUTPUT)
	tools.Register(tools.TASK_STOP, STOP)
}

// Names lists every tool name this package contributes.
func Names() []tools.ToolName { return taskNames }
