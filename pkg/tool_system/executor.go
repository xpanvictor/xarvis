package toolsystem

import (
	"context"
	"fmt"
	"time"

	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

type ToolExecutionResult struct {
	ToolCall *adapters.ContractToolCall
	Result   map[string]any
	Error    error
	Duration time.Duration
}

type Executor interface {
	Execute(ctx context.Context, reg Registry, call adapters.ContractToolCall) (*ToolExecutionResult, error)
}

type executor struct{}

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

	// Metric tracking
	startTime := time.Now()

	// Execute the tool handler
	result, toolErr := selectedTool.Handler(ctx, call.Arguments)
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
	return &executor{}
}
