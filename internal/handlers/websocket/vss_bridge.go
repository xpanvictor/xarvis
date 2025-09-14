package websocket

import (
	"context"
	"time"

	vss "github.com/xpanvictor/xarvis/internal/domains/sys_manager/voice_stream_system"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// VSSEventBridge connects VSS events with WebSocket messages
type VSSEventBridge struct {
	logger            *Logger.Logger
	inputManager      *InputStreamManager
	connectionManager *ConnectionManager
}

// NewVSSEventBridge creates a new VSS event bridge
func NewVSSEventBridge(
	logger *Logger.Logger,
	inputManager *InputStreamManager,
	connectionManager *ConnectionManager,
) *VSSEventBridge {
	return &VSSEventBridge{
		logger:            logger,
		inputManager:      inputManager,
		connectionManager: connectionManager,
	}
}

// HandleVSSEvents listens to VSS output events and processes them
func (bridge *VSSEventBridge) HandleVSSEvents(ctx context.Context, session *Session) {
	if session.VoiceSystem == nil {
		bridge.logger.Errorf("No VSS for session %s", session.SessionID)
		return
	}

	bridge.logger.Infof("Starting VSS event handling for session %s", session.SessionID)

	// Listen to VSS output events
	for {
		select {
		case <-ctx.Done():
			bridge.logger.Infof("VSS event handling stopped for session %s", session.SessionID)
			return

		case vssEvent, ok := <-session.VoiceSystem.GetOutputChannel():
			if !ok {
				bridge.logger.Infof("VSS output channel closed for session %s", session.SessionID)
				return
			}

			bridge.processVSSEvent(ctx, session, vssEvent)
		}
	}
}

// processVSSEvent processes individual VSS events
func (bridge *VSSEventBridge) processVSSEvent(ctx context.Context, session *Session, event vss.VSSEvent) {
	bridge.logger.Debugf("Processing VSS event type: %s for session %s", event.Type, session.SessionID)

	switch event.Type {
	case vss.EventInterrupt:
		bridge.handleVSSInterrupt(ctx, session, event)

	case vss.EventListening:
		bridge.handleListeningModeChange(session, event)

	default:
		bridge.logger.Debugf("Unhandled VSS event type: %s", event.Type)
	}
}

// handleVSSInterrupt processes VSS interrupt events with transcription
func (bridge *VSSEventBridge) handleVSSInterrupt(ctx context.Context, session *Session, event vss.VSSEvent) {
	if interruptData, ok := event.Data.(vss.InterruptData); ok {
		bridge.logger.Infof("VSS interrupt for user %s: %s (confidence: %.2f)",
			session.UserID, interruptData.Transcription, interruptData.Confidence)

		// Process transcribed text as user input
		err := bridge.inputManager.HandleTranscribedText(ctx, session, interruptData.Transcription)
		if err != nil {
			bridge.logger.Errorf("Failed to process VSS interrupt: %v", err)
			session.SendError("TRANSCRIPTION_ERROR", "Failed to process your voice input")
			return
		}

		// Notify VSS that audio processing is done
		doneEvent := vss.VSSEvent{
			Type:      vss.EventAudProcDone,
			UserID:    session.UserID,
			SessionID: session.SessionID,
			Timestamp: time.Now(),
		}

		err = session.SendVSSEvent(doneEvent)
		if err != nil {
			bridge.logger.Errorf("Failed to send AudProcDone to VSS: %v", err)
		}
	} else {
		bridge.logger.Errorf("Invalid interrupt data type for session %s", session.SessionID)
	}
}

// handleListeningModeChange processes VSS listening mode changes
func (bridge *VSSEventBridge) handleListeningModeChange(session *Session, event vss.VSSEvent) {
	bridge.logger.Debugf("Listening mode change for session %s: %v", session.SessionID, event.Data)

	// Extract mode information
	var mode string
	if modeStr, ok := event.Data.(string); ok {
		mode = modeStr
	} else {
		// Fallback: try to extract mode from the event data
		mode = "unknown"
		bridge.logger.Warnf("Unknown listening mode data type for session %s: %T", session.SessionID, event.Data)
	}

	// Send listening mode change to client
	modeData := ListeningStateMessage{
		Mode:      mode,
		Timestamp: event.Timestamp,
	}

	err := session.SendWebSocketMessage(MessageTypeListeningState, modeData)
	if err != nil {
		bridge.logger.Errorf("Failed to send listening mode change: %v", err)
	}
}

// SendNeedMoreContext sends a need more context event to VSS
func (bridge *VSSEventBridge) SendNeedMoreContext(session *Session) error {
	event := vss.VSSEvent{
		Type:      vss.EventNeedMoreCtx,
		UserID:    session.UserID,
		SessionID: session.SessionID,
		Timestamp: time.Now(),
	}

	return session.SendVSSEvent(event)
}
