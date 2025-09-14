package websocket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	vss "github.com/xpanvictor/xarvis/internal/domains/sys_manager/voice_stream_system"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
)

// InputStreamManager handles processing of various input types
type InputStreamManager struct {
	logger              *Logger.Logger
	conversationService conversation.ConversationService
}

// NewInputStreamManager creates a new input stream manager
func NewInputStreamManager(
	logger *Logger.Logger,
	conversationService conversation.ConversationService,
) *InputStreamManager {
	return &InputStreamManager{
		logger:              logger,
		conversationService: conversationService,
	}
}

// HandleTextInput processes text input from the user
func (im *InputStreamManager) HandleTextInput(
	ctx context.Context,
	session *Session,
	text string,
) error {
	// Validate input
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty text input")
	}

	im.logger.Infof("Processing text input from user %s: %s", session.UserID, text)

	// Create message
	message := types.Message{
		Id:        uuid.New(),
		UserId:    session.UserID,
		Text:      text,
		Timestamp: time.Now(),
		MsgRole:   assistant.USER,
		Tags:      []string{"websocket", "text"},
	}

	// Create output channel for streaming responses
	outputCh := make(chan types.Message, 10)
	defer close(outputCh)

	// Start goroutine to handle streaming responses
	go im.handleResponseStream(ctx, session, outputCh)

	// Process through conversation service
	err := im.conversationService.ProcessMsgAsStream(
		ctx,
		session.UserID,
		message,
		[]types.Message{}, // TODO: Add system messages from conversation history
		outputCh,
	)

	if err != nil {
		im.logger.Errorf("Failed to process text input: %v", err)
		session.SendError("PROCESSING_ERROR", "Failed to process your message")
		return err
	}

	return nil
}

// HandleAudioInput processes audio input from the user
func (im *InputStreamManager) HandleAudioInput(
	ctx context.Context,
	session *Session,
	audioData AudioMessage,
) error {
	im.logger.Debugf("Processing audio input from user %s: %d bytes",
		session.UserID, len(audioData.Data))

	// Convert to VSS audio input format
	audioInput := audioring.AudioInput{
		Data:       audioData.Data,
		Timestamp:  time.Now(),
		SampleRate: audioData.SampleRate,
		Channels:   audioData.Channels,
	}

	// Send to VSS using existing event system
	vssEvent := vss.VSSEvent{
		Type:      vss.EventAudioInput,
		UserID:    session.UserID,
		SessionID: session.SessionID,
		Data: vss.AudioInputData{
			AudioInput: audioInput,
		},
		Timestamp: time.Now(),
	}

	err := session.SendVSSEvent(vssEvent)
	if err != nil {
		im.logger.Errorf("Failed to send audio to VSS: %v", err)
		return err
	}

	// Update session activity
	session.UpdateLastActive()
	return nil
}

// HandleListeningControl processes listening control commands
func (im *InputStreamManager) HandleListeningControl(
	ctx context.Context,
	session *Session,
	control ListeningControl,
) error {
	im.logger.Infof("Processing listening control from user %s: %s",
		session.UserID, control.Action)

	var eventType vss.EventType

	switch control.Action {
	case "start_listening":
		eventType = vss.EventResumeListening
	case "stop_listening":
		eventType = vss.EventStopListening
	default:
		return fmt.Errorf("unknown listening control action: %s", control.Action)
	}

	vssEvent := vss.VSSEvent{
		Type:      eventType,
		UserID:    session.UserID,
		SessionID: session.SessionID,
		Timestamp: time.Now(),
	}

	err := session.SendVSSEvent(vssEvent)
	if err != nil {
		im.logger.Errorf("Failed to send listening control to VSS: %v", err)
		return err
	}

	// Update session activity
	session.UpdateLastActive()
	return nil
}

// HandleTranscribedText processes transcribed text from VSS
func (im *InputStreamManager) HandleTranscribedText(
	ctx context.Context,
	session *Session,
	transcription string,
) error {
	// Validate transcription
	transcription = strings.TrimSpace(transcription)
	if transcription == "" {
		im.logger.Debugf("Empty transcription from VSS for user %s", session.UserID)
		return nil
	}

	im.logger.Infof("Processing transcribed text from user %s: %s",
		session.UserID, transcription)

	// Create message with transcribed tag
	message := types.Message{
		Id:        uuid.New(),
		UserId:    session.UserID,
		Text:      transcription,
		Timestamp: time.Now(),
		MsgRole:   assistant.USER,
		Tags:      []string{"websocket", "transcribed", "vss"},
	}

	// Create output channel for streaming responses
	outputCh := make(chan types.Message, 10)
	defer close(outputCh)

	// Start goroutine to handle streaming responses
	go im.handleResponseStream(ctx, session, outputCh)

	// Process through conversation service
	err := im.conversationService.ProcessMsgAsStream(
		ctx,
		session.UserID,
		message,
		[]types.Message{}, // TODO: Add system messages from conversation history
		outputCh,
	)

	if err != nil {
		im.logger.Errorf("Failed to process transcribed text: %v", err)
		session.SendError("PROCESSING_ERROR", "Failed to process your transcribed message")
		return err
	}

	return nil
}

// handleResponseStream handles streaming responses from the conversation service
func (im *InputStreamManager) handleResponseStream(
	ctx context.Context,
	session *Session,
	outputCh <-chan types.Message,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case response, ok := <-outputCh:
			if !ok {
				return
			}

			// Send response to client
			err := session.SendResponse(response.Text, "text")
			if err != nil {
				im.logger.Errorf("Failed to send response to client: %v", err)
				return
			}
		}
	}
}
