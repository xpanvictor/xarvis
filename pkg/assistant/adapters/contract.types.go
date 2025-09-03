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

type ContractToolIOType struct {
	Type  string // obj default
	KVMap map[string]any
}
type ContractToolFn struct {
	Props           ContractToolIOType
	RequiredProps   []string
	OutputStructure ContractToolIOType
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
	Arguments ContractToolIOType // so value will be actual value not type
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
