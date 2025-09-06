package device

import (
	"time"

	"github.com/google/uuid"
)

type Transport string

const (
	TransportMQTT Transport = "mqtt"
	TransportWS   Transport = "ws"
)

type OutputMessageType int

const (
	EText OutputMessageType = iota
	EAudio
	EEvent
)

type Capabilities struct {
	AudioSink bool // can sink audio
	TextSink  bool // can sink audio
}

type EndpointID uuid.UUID

type Endpoint interface {
	// Identity
	ID() EndpointID
	Caps() Capabilities
	Transport() Transport
	// abstraction for publisher
	SendTextDelta(sessionID uuid.UUID, seq int, text string) error
	SendAudioFrame(sessionID uuid.UUID, seq int, frame []byte) error
	SendEvent(sessionID uuid.UUID, name string, payload any) error
	Touch()
	// lifecyle
	IsAlive() bool
	Close() error
	LastActive() time.Time
}

type Device struct {
	UserID     uuid.UUID
	DeviceID   uuid.UUID
	SessionID  uuid.UUID
	Caps       Capabilities
	LastActive time.Time
	// each device can handle multiple endpoints (ws, mqtt, etc)
	Endpoints map[EndpointID]Endpoint
}
