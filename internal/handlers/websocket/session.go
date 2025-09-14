package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	vss "github.com/xpanvictor/xarvis/internal/domains/sys_manager/voice_stream_system"
	"github.com/xpanvictor/xarvis/pkg/io/device"
)

// Session represents a WebSocket session for a user
type Session struct {
	UserID       uuid.UUID
	SessionID    uuid.UUID
	DeviceID     uuid.UUID
	Conn         *websocket.Conn
	Capabilities device.Capabilities

	// VSS Integration (using existing VSS)
	VoiceSystem *vss.VSS
	VSSContext  context.Context
	VSSCancel   context.CancelFunc

	// State
	ConnectedAt time.Time
	lastActive  time.Time // Use lowercase to avoid conflict with method
	IsActive    bool
	mutex       sync.RWMutex
}

// NewSession creates a new WebSocket session
func NewSession(userID uuid.UUID, conn *websocket.Conn, capabilities device.Capabilities) *Session {
	return &Session{
		UserID:       userID,
		SessionID:    uuid.New(),
		DeviceID:     uuid.New(),
		Conn:         conn,
		Capabilities: capabilities,
		ConnectedAt:  time.Now(),
		lastActive:   time.Now(),
		IsActive:     true,
	}
}

// SendVSSEvent sends an event to the VSS system
func (s *Session) SendVSSEvent(event vss.VSSEvent) error {
	if s.VoiceSystem == nil {
		return fmt.Errorf("VSS not initialized")
	}

	select {
	case s.VoiceSystem.GetInputChannel() <- event:
		return nil
	default:
		return fmt.Errorf("VSS input channel full")
	}
}

// SendWebSocketMessage sends a message to the WebSocket client
func (s *Session) SendWebSocketMessage(msgType MessageType, data interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.IsActive {
		return fmt.Errorf("session not active")
	}

	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		SessionID: s.SessionID.String(),
		Timestamp: time.Now(),
	}

	return s.Conn.WriteJSON(msg)
}

// SendError sends an error message to the client
func (s *Session) SendError(code, message string) error {
	return s.SendWebSocketMessage(MessageTypeError, ErrorMessage{
		Code:    code,
		Message: message,
	})
}

// SendResponse sends a response message to the client
func (s *Session) SendResponse(content, responseType string) error {
	return s.SendWebSocketMessage(MessageTypeResponse, ResponseMessage{
		Content:   content,
		Type:      responseType,
		Timestamp: time.Now(),
	})
}

// UpdateLastActive updates the last activity timestamp
func (s *Session) UpdateLastActive() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastActive = time.Now()
}

// Close closes the session and cleans up resources
func (s *Session) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.IsActive = false

	// Cancel VSS context if it exists
	if s.VSSCancel != nil {
		s.VSSCancel()
	}

	// Close VSS if it exists
	if s.VoiceSystem != nil {
		if err := s.VoiceSystem.Close(); err != nil {
			return fmt.Errorf("failed to close VSS: %v", err)
		}
	}

	// Close WebSocket connection
	return s.Conn.Close()
}

// IsExpired checks if the session has expired based on inactivity
func (s *Session) IsExpired(timeout time.Duration) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return time.Since(s.lastActive) > timeout
}

// Implement device.Endpoint interface

// ID returns the endpoint ID
func (s *Session) ID() device.EndpointID {
	return device.EndpointID(s.SessionID)
}

// Caps returns the endpoint capabilities
func (s *Session) Caps() device.Capabilities {
	return s.Capabilities
}

// Transport returns the transport type
func (s *Session) Transport() device.Transport {
	return device.TransportWS
}

// SendTextDelta sends text delta to the WebSocket client
func (s *Session) SendTextDelta(sessionID uuid.UUID, seq int, text string) error {
	return s.SendWebSocketMessage(MessageTypeResponse, ResponseMessage{
		Content:   text,
		Type:      "text_delta",
		Timestamp: time.Now(),
	})
}

// SendAudioFrame sends audio frame to the WebSocket client
func (s *Session) SendAudioFrame(sessionID uuid.UUID, seq int, frame []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.IsActive {
		return fmt.Errorf("session not active")
	}

	// Send binary audio frame
	return s.Conn.WriteMessage(websocket.BinaryMessage, frame)
}

// SendEvent sends event to the WebSocket client
func (s *Session) SendEvent(sessionID uuid.UUID, name string, payload any) error {
	return s.SendWebSocketMessage(MessageTypeResponse, ResponseMessage{
		Content:   fmt.Sprintf("Event: %s", name),
		Type:      "event",
		Timestamp: time.Now(),
	})
}

// Touch updates the last activity timestamp
func (s *Session) Touch() {
	s.UpdateLastActive()
}

// IsAlive checks if the session is active
func (s *Session) IsAlive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.IsActive
}

// LastActive returns the last activity timestamp
func (s *Session) LastActive() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastActive
}
