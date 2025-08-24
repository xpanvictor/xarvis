package assistant

import (
	"context"
	"time"
)

type Role string

const (
	USER      Role = "user"
	ASSISTANT Role = "assistant"
	SYSTEM    Role = "system"
	TOOL      Role = "tool"
)

type AssistantMessage struct {
	Content      string
	CreatedAt    time.Time
	MsgRole      Role
	ToolResponse *ToolCall
}

type AssistantToolType struct {
	Name        string
	Paramters   []interface{}
	Output      []interface{}
	Description string
}

type AssistantToolsElement struct {
	Id              string
	ToolList        []AssistantToolType
	StoredToolCalls []ToolCall
}

type AssistantInput struct {
	Msgs           []AssistantMessage
	Meta           interface{}
	AvailableTools []AssistantToolType
}

type ToolCallStatus int

const (
	PENDING ToolCallStatus = iota
	CONSUMED
	SUCCESS
	PROGRESS
	FAILED
)

type ToolResult struct {
	Response map[string]any
}

type ToolCall struct {
	Id              string
	Name            string
	Arguments       map[string]any
	Status          ToolCallStatus
	Result          *ToolResult
	RunningDuration time.Duration
}

type AssistantOutput struct {
	Id        string
	Response  AssistantMessage
	ToolCalls []ToolCall
}

type Assistant interface {
	ProcessPrompt(ctx context.Context, input AssistantInput) (*AssistantOutput, error)
}
