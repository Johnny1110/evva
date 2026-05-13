package tools

import "fmt"

// Registry maps tool names to Tool implementations. .
type Registry struct {
	tools map[ToolName]Tool
}

var StaticToolRegistry = &Registry{
	tools: make(map[ToolName]Tool),
}

// if create a new tool, mast register into this Registry.
func Register(toolName ToolName, tool Tool) {
	StaticToolRegistry.tools[toolName] = tool
}

// GetTools expose ToolRegistry for agent.Agent
func GetTools(tools ...ToolName) (map[string]Tool, error) {
	toolMap := make(map[string]Tool)
	if len(tools) == 0 {
		return toolMap, nil
	}

	for _, toolName := range tools {
		instance, ok := StaticToolRegistry.tools[toolName]
		if !ok {
			return nil, fmt.Errorf("tool %s not found in StaticToolRegistry", toolName)
		}
		toolMap[string(toolName)] = instance
	}

	return toolMap, nil
}

func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[ToolName(name)]
	if !ok {
		return nil, fmt.Errorf("tool %q not registered", name)
	}
	return t, nil
}

func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}
