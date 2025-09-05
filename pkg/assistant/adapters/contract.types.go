package adapters

import (
	"time"

	"github.com/google/uuid"
)

type MsgRole string

const (
	USER      MsgRole = "user"
	ASSISTANT MsgRole = "assistant"
	SYSTEM    MsgRole = "system"
	TOOL      MsgRole = "tool"
)

type ContractMessage struct {
	Role      MsgRole
	Content   string
	CreatedAt time.Time
}

type ContractToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type ContractToolIOType struct {
	Type       string                          `json:"type"` // "object" default
	Properties map[string]ContractToolProperty `json:"properties"`
}

type ContractToolFn struct {
	Parameters      ContractToolIOType `json:"parameters"`
	RequiredProps   []string           `json:"required"`
	OutputStructure ContractToolIOType `json:"output_structure"`
}
type ContractTool struct {
	Name         string
	Type         string // function default
	Description  string
	ToolFunction ContractToolFn
}

type ContractSelectedModel struct {
	Name    string
	Version string
}

type ContractInput struct {
	ID           uuid.UUID
	ToolList     []ContractTool
	Meta         any // json serialized
	Msgs         []ContractMessage
	HandlerModel ContractSelectedModel
}

type ContractToolCall struct {
	ID        uuid.UUID
	CreatedAt time.Time
	ToolName  string
	Arguments map[string]any // actual values, not type definitions
}

// response is by default a stream
type ContractResponseDelta struct {
	Msg       *ContractMessage
	ToolCalls *[]ContractToolCall
	Error     error
	Index     uint
	Done      bool
	CreatedAt time.Time
}

type ContractResponseChannel chan []ContractResponseDelta

type ContractResponse struct {
	ID        uuid.UUID
	StartedAt time.Time
	Error     error
	Done      bool
}
