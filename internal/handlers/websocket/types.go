package websocket

import (
	"time"

	"github.com/xpanvictor/xarvis/pkg/io/device"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	MessageTypeText             MessageType = "text"
	MessageTypeAudio            MessageType = "audio"
	MessageTypeInit             MessageType = "init"
	MessageTypeListeningState   MessageType = "listening_state"
	MessageTypeListeningControl MessageType = "listening_control"
	MessageTypeResponse         MessageType = "response"
	MessageTypeError            MessageType = "error"
)

// WSMessage represents the structure of WebSocket messages
type WSMessage struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
	Sequence  int         `json:"sequence,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// AudioMessage contains audio data payload
type AudioMessage struct {
	SampleRate int32  `json:"sampleRate"`
	Channels   int16  `json:"channels"`
	Data       []byte `json:"data"`
}

// TextMessage contains text input payload
type TextMessage struct {
	Content string `json:"content"`
}

// InitMessage contains initialization data
type InitMessage struct {
	Capabilities device.Capabilities `json:"capabilities"`
	UserID       string              `json:"userId,omitempty"`
}

// ListeningControl contains listening control commands
type ListeningControl struct {
	Action string `json:"action"` // "start_listening", "stop_listening"
}

// ListeningStateMessage contains listening state information
type ListeningStateMessage struct {
	Mode      string    `json:"mode"` // "active", "passive"
	Timestamp time.Time `json:"timestamp"`
}

// ErrorMessage contains error information
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResponseMessage contains system response information
type ResponseMessage struct {
	Content   string    `json:"content"`
	Type      string    `json:"type"` // "text", "audio"
	Timestamp time.Time `json:"timestamp"`
}
