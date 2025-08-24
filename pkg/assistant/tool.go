package assistant

import (
	"fmt"
)

func NewAssistantTool(
	name string,
	parameters []interface{},
	output []interface{},
	description string,
) AssistantToolType {
	return AssistantToolType{
		Name:        name,
		Paramters:   parameters,
		Output:      output,
		Description: description,
	}
}

func (ate *AssistantToolsElement) CallTool(
	name string,
	args map[string]any,
) (*ToolCall, error) {
	for _, tool := range ate.ToolList {
		if tool.Name == name {
			toolCall := ToolCall{
				Id:        "todo",
				Name:      tool.Name,
				Arguments: args,
				Status:    PENDING,
			}
			ate.StoredToolCalls = append(ate.StoredToolCalls, toolCall)
			return &toolCall, nil
		}
	}
	return nil, fmt.Errorf("tool call not found: `%v`", name)
}

func (ate *AssistantToolsElement) ConsumeCall() (*ToolCall, bool) {
	if len(ate.StoredToolCalls) <= 0 {
		return nil, false
	}

	toolCall := ate.StoredToolCalls[0]
	ate.StoredToolCalls = ate.StoredToolCalls[1:]
	return &toolCall, true
}
