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
)

type AssistantMessage struct {
	Content   string
	CreatedAt time.Time
	MsgRole   Role
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
	FAILED
)

type ToolCall struct {
	Id        string
	Name      string
	Arguments []interface{}
	Status    ToolCallStatus
}

type AssistantOutput struct {
	Id        string
	Response  AssistantMessage
	ToolCalls []ToolCall
}

type Assistant interface {
	ProcessPrompt(ctx context.Context, input AssistantInput) (*AssistantOutput, error)
}
