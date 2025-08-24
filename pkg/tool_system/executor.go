package toolsystem

import (
	"context"
	"fmt"
	"time"

	"github.com/xpanvictor/xarvis/pkg/assistant"
)

type Executor interface {
	Execute(ctx context.Context, reg Registry, call assistant.ToolCall) (*assistant.ToolCall, *assistant.ToolResult, error)
}

type executor struct{}

// Execute implements Executor.
func (e *executor) Execute(ctx context.Context, reg Registry, call assistant.ToolCall) (*assistant.ToolCall, *assistant.ToolResult, error) {
	// retrieve tool from registry
	tool, ok := reg.Get(call.Name)
	if !ok {
		return nil, nil, fmt.Errorf("tool not found")
	}
	// todo: validate input
	// metric
	startTime := time.Now()
	// todo: check if async action

	// handle fn
	res, toolErr := tool.Handler(ctx, call.Arguments)
	runningDuration := time.Since(startTime)

	if toolErr != nil {
		call.Status = assistant.FAILED
		call.Result = &assistant.ToolResult{
			Response: map[string]any{
				"error": toolErr.Error(),
			},
		}
		call.RunningDuration = runningDuration

		return &call, call.Result, toolErr
	}

	toolResult := assistant.ToolResult{
		Response: res,
	}

	// Update the call with success status and result
	call.Status = assistant.SUCCESS
	call.Result = &toolResult
	call.RunningDuration = runningDuration

	return &call, &toolResult, nil
}

func (e *executor) AsyncExecute() {}

func NewExecutor() Executor {
	return &executor{}
}
