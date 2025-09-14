package toolsystem

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

// UserContext contains user metadata for tool execution
type UserContext struct {
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	UserEmail       string    `json:"user_email"`
	CurrentDateTime time.Time `json:"current_date_time"`
}

type ToolExecutionResult struct {
	ToolCall *adapters.ContractToolCall
	Result   map[string]any
	Error    error
	Duration time.Duration
}

type Executor interface {
	Execute(ctx context.Context, reg Registry, call adapters.ContractToolCall) (*ToolExecutionResult, error)
	SetUserContext(userCtx *UserContext) // Set user context for this execution session
}

type executor struct {
	userContext *UserContext
}

// SetUserContext implements Executor.
func (e *executor) SetUserContext(userCtx *UserContext) {
	e.userContext = userCtx
}

// Execute implements Executor.
func (e *executor) Execute(ctx context.Context, reg Registry, call adapters.ContractToolCall) (*ToolExecutionResult, error) {
	// retrieve tool from registry by name
	var selectedTool Tool
	var found bool

	// Search through all tools to find one with matching name
	tools := reg.List()
	for _, tool := range tools {
		if tool.Spec.Name == call.ToolName {
			selectedTool = tool
			found = true
			break
		}
	}

	if !found {
		return &ToolExecutionResult{
			ToolCall: &call,
			Error:    fmt.Errorf("tool not found: %s", call.ToolName),
		}, fmt.Errorf("tool not found: %s", call.ToolName)
	}

	// Inject user context into the arguments if executor has user context set
	args := call.Arguments
	if e.userContext != nil {
		// Clone the arguments map to avoid modifying the original
		args = make(map[string]any)
		for k, v := range call.Arguments {
			args[k] = v
		}

		// Inject meta context (these override any existing values for security)
		args["__user_id"] = e.userContext.UserID.String()
		args["__username"] = e.userContext.Username
		args["__user_email"] = e.userContext.UserEmail
		args["__current_date_time"] = e.userContext.CurrentDateTime.Format(time.RFC3339)
	}

	// Metric tracking
	startTime := time.Now()

	// Execute the tool handler with injected context
	result, toolErr := selectedTool.Handler(ctx, args)
	runningDuration := time.Since(startTime)

	// Create the execution result
	executionResult := &ToolExecutionResult{
		ToolCall: &call,
		Result:   result,
		Error:    toolErr,
		Duration: runningDuration,
	}

	return executionResult, toolErr
}

func (e *executor) AsyncExecute() {}

func NewExecutor() Executor {
	return &executor{
		userContext: nil, // Will be set via SetUserContext before execution
	}
}
