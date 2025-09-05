package toolsystem

import (
	"context"

	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

// ToolHandler function signature for tool execution
type ToolHandler func(ctx context.Context, args map[string]any) (map[string]any, error)

// ConvertToolSpecToContract converts a tool spec to contract format
func ConvertToolSpecToContract(name, version, description string, properties map[string]adapters.ContractToolProperty, required []string) adapters.ContractTool {
	return adapters.ContractTool{
		Name:        name,
		Type:        "function", // default type
		Description: description,
		ToolFunction: adapters.ContractToolFn{
			Parameters: adapters.ContractToolIOType{
				Type:       "object",
				Properties: properties,
			},
			RequiredProps: required,
			// OutputStructure can be defined later if needed
			OutputStructure: adapters.ContractToolIOType{
				Type:       "object",
				Properties: make(map[string]adapters.ContractToolProperty),
			},
		},
	}
}
