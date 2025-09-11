package voicestreamsystem

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
	"github.com/xpanvictor/xarvis/pkg/io/stt/vad"
	"github.com/xpanvictor/xarvis/pkg/io/stt/whisper"
)

// VSSMode represents the current listening mode
type VSSMode int

const (
	PassiveListening VSSMode = iota
	ActiveListening
)

// Event types for VSS communication
type EventType string

const (
	// Input events (commands to VSS)
	EventAudioInput      EventType = "AUDIO_INPUT"
	EventAudProcDone     EventType = "AUD_PROC_DONE"
	EventNeedMoreCtx     EventType = "NEED_MORE_CTX"
	EventStopListening   EventType = "STOP_LISTENING"
	EventResumeListening EventType = "RESM_LISTENING"

	// Output events (from VSS)
	EventInterrupt EventType = "INTR"
	EventListening EventType = "listening_mode_change"
)

// VSSEvent represents communication with the VSS
type VSSEvent struct {
	Type      EventType   `json:"type"`
	UserID    uuid.UUID   `json:"userId"`
	SessionID uuid.UUID   `json:"sessionId"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// AudioInputData contains audio input information
type AudioInputData struct {
	AudioInput audioring.AudioInput `json:"audioInput"`
}

// InterruptData contains transcription and timing information
type InterruptData struct {
	Transcription string         `json:"transcription"`
	StartTime     time.Time      `json:"startTime"`
	EndTime       time.Time      `json:"endTime"`
	Confidence    float64        `json:"confidence"`
	Segments      []TimedSegment `json:"segments,omitempty"`
}

// TimedSegment represents a timed segment of transcription
type TimedSegment struct {
	Text      string    `json:"text"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// VSSConfig contains configuration for the VSS
type VSSConfig struct {
	BufferCapacity         int           `json:"bufferCapacity"`         // Max audio frames to buffer
	SilenceThreshold       time.Duration `json:"silenceThreshold"`       // Silence threshold for passive mode
	ActiveSilenceThreshold time.Duration `json:"activeSilenceThreshold"` // Silence threshold for active mode
	VADConfig              vad.VADConfig `json:"vadConfig"`              // VAD configuration
}

// DefaultVSSConfig returns default configuration
func DefaultVSSConfig() VSSConfig {
	return VSSConfig{
		BufferCapacity:         100, // Max 100 audio frames
		SilenceThreshold:       2 * time.Second,
		ActiveSilenceThreshold: 700 * time.Millisecond,
		VADConfig:              vad.DefaultVADConfig(),
	}
}

// VSS represents the Voice Streaming System deployed per user
type VSS struct {
	userID    uuid.UUID
	sessionID uuid.UUID
	config    VSSConfig
	logger    *Logger.Logger

	// Core components
	audioBuffer   []audioring.AudioInput // Simple slice buffer for all audio frames
	vad           vad.VAD                // Voice Activity Detection (for future use)
	whisperClient *whisper.WhisperClient // Whisper STT client
	mode          VSSMode
	isProcessing  bool

	// Channels for communication
	inCh  chan VSSEvent
	outCh chan VSSEvent

	// State management
	mutex               sync.RWMutex
	listeningTimer      time.Timer
	transcriptionBuffer []whisper.TranscriptionResponse
}

// NewVSS creates a new Voice Streaming System instance
func NewVSS(userID, sessionID uuid.UUID, vssConfig VSSConfig, appConfig *config.Settings, logger *Logger.Logger) *VSS {
	// Use configured sys-models URL, fallback to localhost for development
	sysModelsURL := appConfig.SysModels.BaseURL
	if appConfig.SysModels.BaseURL != "" {
		sysModelsURL = appConfig.SysModels.BaseURL
	}

	// Initialize VAD (currently commented out but kept for future use)
	vadInstance := vad.NewSileroVADWithURL(vssConfig.VADConfig, logger, sysModelsURL)

	// Initialize Whisper client
	whisperClient := whisper.NewWhisperClient(appConfig.Voice.STTURL, logger)

	return &VSS{
		userID:        userID,
		sessionID:     sessionID,
		config:        vssConfig,
		logger:        logger,
		audioBuffer:   make([]audioring.AudioInput, 0, vssConfig.BufferCapacity),
		vad:           vadInstance,
		whisperClient: whisperClient,
		mode:          PassiveListening,
		inCh:          make(chan VSSEvent, 1000), // Larger buffer for high-frequency audio
		outCh:         make(chan VSSEvent, 10),   // Buffered channel for outputs
	}
}

// GetInputChannel returns the input channel for sending events to VSS
func (v *VSS) GetInputChannel() chan<- VSSEvent {
	return v.inCh
}

// GetOutputChannel returns the output channel for receiving events from VSS
func (v *VSS) GetOutputChannel() <-chan VSSEvent {
	return v.outCh
}

// Run starts the VSS processing loop
func (v *VSS) Run(ctx context.Context) {
	// Ticker for processing and emptying buffer every n seconds
	processTicker := time.NewTicker(2 * time.Second)
	defer processTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(v.outCh)
			return

		case event := <-v.inCh:
			// Process audio events immediately to prevent channel backup
			v.handleEvent(ctx, event)

		case <-processTicker.C:
			v.processBufferedAudio(ctx)

		case <-v.listeningTimer.C:
			if v.mode == ActiveListening {
				// process
				v.sendTranscriptionInterrupt()
				v.SetMode(PassiveListening)
				v.listeningTimer.Stop()
			}
		}
	}
}

// handleEvent processes incoming events
func (v *VSS) handleEvent(ctx context.Context, event VSSEvent) {
	switch event.Type {
	case EventAudioInput:
		v.handleAudioInput(event)

	case EventAudProcDone:
		v.handleAudProcDone()

	case EventNeedMoreCtx:
		v.handleNeedMoreCtx()

	case EventStopListening:
		v.handleStopListening()

	case EventResumeListening:
		v.handleResumeListening()

	default:
		v.logger.Warnf("Unknown event type %s for user %s", event.Type, v.userID)
	}
}

// handleAudioInput processes incoming audio data
func (v *VSS) handleAudioInput(event VSSEvent) {
	if audioData, ok := event.Data.(AudioInputData); ok {
		v.mutex.Lock()
		defer v.mutex.Unlock()

		// Push all audio to buffer (for now, every 5s we'll process it)
		// TODO: In future, only push after VAD detects voice activity
		// vadResult, err := v.vad.DetectVoice(ctx, audioData.AudioInput)
		// if err == nil && vadResult.HasVoice { ... }

		// Add to buffer
		if len(v.audioBuffer) >= v.config.BufferCapacity {
			// Remove oldest frame if buffer is full to prevent overflow
			v.audioBuffer = v.audioBuffer[1:]
		}
		v.audioBuffer = append(v.audioBuffer, audioData.AudioInput)
	}
}

// processBufferedAudio processes and empties the audio buffer every 5 seconds
func (v *VSS) processBufferedAudio(ctx context.Context) {
	v.mutex.Lock()

	if len(v.audioBuffer) == 0 {
		v.mutex.Unlock()
		return
	}

	// Copy buffer and clear it
	audioFrames := make([]audioring.AudioInput, len(v.audioBuffer))
	copy(audioFrames, v.audioBuffer)
	v.audioBuffer = v.audioBuffer[:0] // Clear buffer

	v.mutex.Unlock()

	v.logger.Infof("Processing %d audio frames for user %s", len(audioFrames), v.userID)

	// Transcribe using Whisper
	transcription, err := v.whisperClient.TranscribeAudio(ctx, audioFrames)
	if err != nil {
		v.logger.Errorf("Failed to transcribe audio for user %s: %v", v.userID, err)
		return
	}

	// Log transcription
	if transcription.Text != "" {
		v.logger.Infof("Transcription for user %s: %s", v.userID, transcription.Text)

		// Send interrupt with transcription
		// v.sendTranscriptionInterrupt(transcription, audioFrames)
		v.HandleTranscription(*transcription)
	} else {
		v.logger.Debugf("Empty transcription for user %s", v.userID)
	}
}

// Checks mode; Decides mode; buffers transcription
func (v *VSS) HandleTranscription(tscp whisper.TranscriptionResponse) {
	if v.mode == PassiveListening {
		// todo: use model to check if transcription is worth listening
		// for now just word search
		v.logger.Infof("checking for word in xarvis: %v", tscp.Text)
		if postTscp := strings.SplitN(tscp.Text, "xarvis", 2); len(postTscp) > 1 {
			postStr := postTscp[1]
			// change to active mode
			v.SetMode(ActiveListening)
			v.logger.Infof("sent active mode")
			v.listeningTimer = *time.NewTimer(v.config.SilenceThreshold)
			// discard buffer
			v.transcriptionBuffer = v.transcriptionBuffer[:0]
			// push last transcription
			tscp.Text = postStr
			v.transcriptionBuffer = append(v.transcriptionBuffer, tscp)
		}
	} else {
		// if new trsc is less than word timer, extend timer
		lastTscpGT := time.Now()
		if tl := len(v.transcriptionBuffer); tl > 0 {
			lastTscpGT = v.transcriptionBuffer[tl-1].GeneratedAt
		}
		if tscp.GeneratedAt.Sub(lastTscpGT) > v.config.ActiveSilenceThreshold {
			v.listeningTimer.Reset(v.config.SilenceThreshold)
		}
		// just append new transcription
		v.transcriptionBuffer = append(v.transcriptionBuffer, tscp)
	}
}

// sendTranscriptionInterrupt sends an interrupt event with the transcription
func (v *VSS) sendTranscriptionInterrupt() {

	v.logger.Infof("Interrupt ------------")
	if len(v.transcriptionBuffer) == 0 {
		return
	}

	tscTextBuilder := strings.Builder{}
	for _, t := range v.transcriptionBuffer {
		tscTextBuilder.WriteString(t.Text)
	}
	transTxt := tscTextBuilder.String()

	// Create interrupt event
	interruptEvent := VSSEvent{
		Type:      EventInterrupt,
		UserID:    v.userID,
		SessionID: v.sessionID,
		Data: InterruptData{
			Transcription: transTxt,
			StartTime:     v.transcriptionBuffer[0].GeneratedAt,
			EndTime:       time.Now(),
			Confidence:    0.95, // Whisper typically has high confidence
		},
		Timestamp: time.Now(),
	}

	// Send interrupt event
	select {
	case v.outCh <- interruptEvent:
		v.logger.Infof("Sent interrupt for user %s: %s", v.userID, transTxt)
	default:
		v.logger.Warnf("Output channel full, dropping interrupt for user %s", v.userID)
	}
	// clear transcription buffer
	v.transcriptionBuffer = v.transcriptionBuffer[:0]
}

func (v *VSS) sendMode() {
	modeEvent := VSSEvent{
		Type:   EventListening,
		UserID: v.userID,
		Data:   v.getModeString(),
	}

	select {
	case v.outCh <- modeEvent:
		v.logger.Infof("send mode to user: %v", modeEvent.Data)
	default:
		v.logger.Info("failed to send mode to user")
	}
}

// Command handlers
func (v *VSS) handleAudProcDone() {
	v.isProcessing = false
}

func (v *VSS) handleNeedMoreCtx() {
	// TODO: Implement context gathering logic
}

func (v *VSS) handleStopListening() {
	// TODO: Implement stop listening logic
}

func (v *VSS) handleResumeListening() {
	// TODO: Implement resume listening logic
}

// Utility methods
func (v *VSS) getModeString() string {
	switch v.mode {
	case PassiveListening:
		return "passive"
	case ActiveListening:
		return "active"
	default:
		return "unknown"
	}
}

// SetMode changes the VSS mode
func (v *VSS) SetMode(mode VSSMode) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	oldModeStr := v.getModeString()
	v.mode = mode
	newModeStr := v.getModeString()

	v.logger.Infof("VSS mode changed for user %s: %s -> %s",
		v.userID, oldModeStr, newModeStr)
	v.sendMode()
}

// GetStats returns current VSS statistics
func (v *VSS) GetStats() map[string]interface{} {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return map[string]interface{}{
		"userID":         v.userID,
		"sessionID":      v.sessionID,
		"mode":           v.getModeString(),
		"bufferLength":   len(v.audioBuffer),
		"bufferCapacity": v.config.BufferCapacity,
		"isProcessing":   v.isProcessing,
	}
}

// Close cleans up VSS resources
func (v *VSS) Close() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Close VAD
	if v.vad != nil {
		if err := v.vad.Close(); err != nil {
			v.logger.Errorf("Failed to close VAD for user %s: %v", v.userID, err)
		}
	}

	// Clear buffer
	v.audioBuffer = nil

	v.logger.Infof("VSS closed for user %s", v.userID)
	return nil
}
