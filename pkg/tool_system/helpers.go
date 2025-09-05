package toolsystem

import (
	"fmt"

	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

// ToolBuilder helps create tools with a fluent interface
type ToolBuilder struct {
	name        string
	version     string
	description string
	properties  map[string]adapters.ContractToolProperty
	required    []string
	handler     ToolHandler
	tags        []string
}

// NewToolBuilder creates a new tool builder
func NewToolBuilder(name, version, description string) *ToolBuilder {
	return &ToolBuilder{
		name:        name,
		version:     version,
		description: description,
		properties:  make(map[string]adapters.ContractToolProperty),
		required:    make([]string, 0),
		tags:        make([]string, 0),
	}
}

// AddParameter adds a parameter to the tool
func (tb *ToolBuilder) AddParameter(name, paramType, description string, required bool, enum ...string) *ToolBuilder {
	tb.properties[name] = adapters.ContractToolProperty{
		Type:        paramType,
		Description: description,
		Enum:        enum,
	}
	if required {
		tb.required = append(tb.required, name)
	}
	return tb
}

// AddStringParameter adds a string parameter
func (tb *ToolBuilder) AddStringParameter(name, description string, required bool, enum ...string) *ToolBuilder {
	return tb.AddParameter(name, "string", description, required, enum...)
}

// AddNumberParameter adds a number parameter
func (tb *ToolBuilder) AddNumberParameter(name, description string, required bool) *ToolBuilder {
	return tb.AddParameter(name, "number", description, required)
}

// AddBooleanParameter adds a boolean parameter
func (tb *ToolBuilder) AddBooleanParameter(name, description string, required bool) *ToolBuilder {
	return tb.AddParameter(name, "boolean", description, required)
}

// AddObjectParameter adds an object parameter
func (tb *ToolBuilder) AddObjectParameter(name, description string, required bool) *ToolBuilder {
	return tb.AddParameter(name, "object", description, required)
}

// AddArrayParameter adds an array parameter
func (tb *ToolBuilder) AddArrayParameter(name, description string, required bool) *ToolBuilder {
	return tb.AddParameter(name, "array", description, required)
}

// SetHandler sets the tool handler function
func (tb *ToolBuilder) SetHandler(handler ToolHandler) *ToolBuilder {
	tb.handler = handler
	return tb
}

// AddTags adds tags to the tool
func (tb *ToolBuilder) AddTags(tags ...string) *ToolBuilder {
	tb.tags = append(tb.tags, tags...)
	return tb
}

// Build creates the final Tool
func (tb *ToolBuilder) Build() (Tool, error) {
	if tb.handler == nil {
		return Tool{}, fmt.Errorf("handler is required for tool %s", tb.name)
	}

	contractTool := ConvertToolSpecToContract(tb.name, tb.version, tb.description, tb.properties, tb.required)

	return Tool{
		Spec:    contractTool,
		Handler: tb.handler,
		Version: tb.version,
		Tags:    tb.tags,
	}, nil
}

// BuildAndRegister creates the tool and registers it to the registry
func (tb *ToolBuilder) BuildAndRegister(registry Registry) error {
	tool, err := tb.Build()
	if err != nil {
		return err
	}
	return registry.Register(tool)
}

// Utility function to create a simple tool quickly
func CreateSimpleTool(name, version, description string, handler ToolHandler, params map[string]ToolParam) (Tool, error) {
	builder := NewToolBuilder(name, version, description).SetHandler(handler)

	for paramName, param := range params {
		builder.AddParameter(paramName, param.Type, param.Description, param.Required, param.Enum...)
	}

	return builder.Build()
}

// ToolParam represents a tool parameter configuration
type ToolParam struct {
	Type        string
	Description string
	Required    bool
	Enum        []string
}
