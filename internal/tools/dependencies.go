package tools

import (
	"fmt"

	"github.com/xpanvictor/xarvis/internal/domains/note"
	"github.com/xpanvictor/xarvis/internal/domains/project"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// ToolDependencies holds all the services and repositories that tools need access to
type ToolDependencies struct {
	// Services
	UserService    user.UserService
	ProjectService project.ProjectService
	NoteService    note.NoteService
	TaskService    task.TaskService

	// Repositories
	ConversationRepo types.ConversationRepository

	// System components
	Logger *Logger.Logger

	// External API keys/configs
	TavilyAPIKey string
}

// ToolFactory creates tools with dependencies injected
type ToolFactory struct {
	deps     *ToolDependencies
	builders map[string]ToolBuilder
	tools    map[string]toolsystem.Tool
}

// NewToolFactory creates a new tool factory with dependencies
func NewToolFactory(deps *ToolDependencies) *ToolFactory {
	return &ToolFactory{
		deps:     deps,
		builders: make(map[string]ToolBuilder),
		tools:    make(map[string]toolsystem.Tool),
	}
}

// GetDependencies returns the tool dependencies
func (tf *ToolFactory) GetDependencies() *ToolDependencies {
	return tf.deps
}

// ToolBuilder interface for tools that need dependencies
type ToolBuilder interface {
	Build(deps *ToolDependencies) (toolsystem.Tool, error)
}

// RegisterBuilder registers a tool builder with a name
func (tf *ToolFactory) RegisterBuilder(name string, builder ToolBuilder) error {
	if _, exists := tf.builders[name]; exists {
		return fmt.Errorf("tool builder '%s' already registered", name)
	}
	tf.builders[name] = builder
	return nil
}

// BuildTool builds a specific tool by name
func (tf *ToolFactory) BuildTool(name string) (toolsystem.Tool, error) {
	builder, exists := tf.builders[name]
	if !exists {
		return toolsystem.Tool{}, fmt.Errorf("tool builder '%s' not found", name)
	}

	tool, err := builder.Build(tf.deps)
	if err != nil {
		return toolsystem.Tool{}, fmt.Errorf("failed to build tool '%s': %w", name, err)
	}

	tf.tools[name] = tool
	return tool, nil
}

// BuildAllTools builds all registered tools
func (tf *ToolFactory) BuildAllTools() (map[string]toolsystem.Tool, error) {
	for name := range tf.builders {
		if _, err := tf.BuildTool(name); err != nil {
			return nil, err
		}
	}
	return tf.tools, nil
}

// GetTools returns all built tools
func (tf *ToolFactory) GetTools() map[string]toolsystem.Tool {
	return tf.tools
}

// RegisterAllTools registers all available tools with their dependencies
// This function should be called from brain system setup to register tools
func RegisterAllTools(deps *ToolDependencies) (*ToolFactory, error) {
	factory := NewToolFactory(deps)

	// Note: Import cycle prevented - tool builders will be registered in brain system setup
	// where catalog package can be imported safely

	return factory, nil
}
