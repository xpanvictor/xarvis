package websockete

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xpanvictor/xarvis/pkg/io/device"
)

type wsEndpoint struct {
	id         uuid.UUID
	client     *websocket.Conn
	caps       device.Capabilities
	lastActive time.Time
}

// Caps implements device.Endpoint.
func (w *wsEndpoint) Caps() device.Capabilities {
	return w.caps
}

// Close implements device.Endpoint.
func (w *wsEndpoint) Close() error {
	return w.client.Close()
}

// ID implements device.Endpoint.
func (w *wsEndpoint) ID() device.EndpointID {
	return device.EndpointID(w.id)
}

func (w *wsEndpoint) Touch() {
	w.lastActive = time.Now()
}

// IsAlive implements device.Endpoint.
// todo: uses ping pong approach
func (w *wsEndpoint) IsAlive() bool {
	// send a ping message
	err := w.client.WriteMessage(9, []byte("ping"))
	return err == nil
}

// LastActive implements device.Endpoint.
func (w *wsEndpoint) LastActive() time.Time {
	return w.lastActive
}

// SendAudioFrame implements device.Endpoint.
func (w *wsEndpoint) SendAudioFrame(sessionID uuid.UUID, seq int, frame []byte) error {
	// Send audio frame metadata as JSON first
	metadata := struct {
		Type      string `json:"type"`
		Index     int    `json:"index"`
		SessionID string `json:"sessionId"`
		Size      int    `json:"size"`
	}{
		Type:      "audio_meta",
		Index:     seq,
		SessionID: sessionID.String(),
		Size:      len(frame),
	}

	// Send metadata as JSON
	if err := w.client.WriteJSON(metadata); err != nil {
		return err
	}

	// Send actual audio data as binary message immediately after
	return w.client.WriteMessage(websocket.BinaryMessage, frame)
}

// SendEvent implements device.Endpoint.
func (w *wsEndpoint) SendEvent(sessionID uuid.UUID, name string, payload any) error {
	msg := struct {
		Name    string `json:"name"`
		Payload any    `json:"payload"`
	}{
		Name:    name,
		Payload: payload,
	}
	return w.client.WriteJSON(msg)
}

// SendTextDelta implements device.Endpoint.
func (w *wsEndpoint) SendTextDelta(sessionID uuid.UUID, seq int, text string) error {
	msg := struct {
		Index int    `json:"index"`
		Text  string `json:"text"`
	}{Index: seq, Text: text}
	return w.client.WriteJSON(msg)
}

// Transport implements device.Endpoint.
func (w *wsEndpoint) Transport() device.Transport {
	return device.TransportWS
}

func New(client *websocket.Conn, caps device.Capabilities) device.Endpoint {
	return &wsEndpoint{
		id:         uuid.New(),
		client:     client,
		caps:       caps,
		lastActive: time.Now(),
	}
}
