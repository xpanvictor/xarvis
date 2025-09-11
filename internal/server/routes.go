package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/conversation/brain"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
	vss "github.com/xpanvictor/xarvis/internal/domains/sys_manager/voice_stream_system"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	olp "github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io"
	"github.com/xpanvictor/xarvis/pkg/io/device"
	websockete "github.com/xpanvictor/xarvis/pkg/io/device/websocket"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	memoryregistry "github.com/xpanvictor/xarvis/pkg/io/registry/memoryRegistry"
	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper/stream"
)

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeAudio MessageType = "audio"
	MessageTypeInit  MessageType = "init"
)

// WebSocket message structure
type WSMessage struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
	Sequence  int         `json:"sequence,omitempty"`
}

// Audio message payload
type AudioMessage struct {
	SampleRate int32  `json:"sampleRate"`
	Channels   int16  `json:"channels"`
	Data       []byte `json:"data"` // PCM frames
}

// Text message payload
type TextMessage struct {
	Content string `json:"content"`
}

// Init message payload
type InitMessage struct {
	Capabilities device.Capabilities `json:"capabilities"`
	UserID       string              `json:"userId,omitempty"`
}

// Connection state for each user
type UserConnection struct {
	UserID        uuid.UUID
	SessionID     uuid.UUID
	DeviceID      uuid.UUID
	Conn          *websocket.Conn
	BrainSystem   *brain.BrainSystem
	AudioBuffer   audioring.AudioRingBuffer
	VoiceSystem   *vss.VSS
	VSSContext    context.Context
	VSSCancel     context.CancelFunc
	TextEndpoint  device.Endpoint
	AudioEndpoint device.Endpoint
	ConnectedAt   time.Time
	LastActive    time.Time
	IsActive      bool
	mutex         sync.RWMutex
}

func (uc *UserConnection) SendEvent(ctx context.Context, ev pipeline.GenericEvent) {
	uc.BrainSystem.Pipeline.SendEvent(ctx, uc.UserID, uc.SessionID, ev)
}

type Dependencies struct {
	conversationRepository conversation.ConversationRepository
	// New brain system dependencies
	DeviceRegistry registry.DeviceRegistry
	Mux            *router.Mux
	BrainConfig    config.BrainConfig
	Logger         *Logger.Logger
	Configs        *config.Settings
}

// UserBrainSystem tracks brain systems per user/session
type UserBrainSystem struct {
	brainSystem *brain.BrainSystem
	userID      uuid.UUID
	sessionID   uuid.UUID
	connectedAt time.Time
}

// RoutesManager manages routes and user sessions
type RoutesManager struct {
	deps            Dependencies
	userConnections map[uuid.UUID]*UserConnection // Track active connections per user
	connectionMutex sync.RWMutex
}

func NewServerDependencies(
	conversationRepository conversation.ConversationRepository,
	deviceRegistry registry.DeviceRegistry,
	mux *router.Mux,
	brainConfig config.BrainConfig,
	logger *Logger.Logger,
	config *config.Settings,
) Dependencies {
	return Dependencies{
		conversationRepository: conversationRepository,
		DeviceRegistry:         deviceRegistry,
		Mux:                    mux,
		BrainConfig:            brainConfig,
		Configs:                config,
		Logger:                 logger,
	}
}

func NewRoutesManager(deps Dependencies) *RoutesManager {
	return &RoutesManager{
		deps:            deps,
		userConnections: make(map[uuid.UUID]*UserConnection),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev-only
}

func InitializeRoutes(cfg *config.Settings, r *gin.Engine, dep Dependencies) {
	r.GET("/", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"message": "Server healthy"}) })
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"status": "ok"}) })

	// Create routes manager with new brain system architecture
	rm := NewRoutesManager(dep)

	// New WebSocket endpoint with audio/text support
	r.GET("/ws", rm.handleWebSocket)

	// Audio-specific endpoint (for dedicated audio streaming)
	r.GET("/ws/audio", rm.handleAudioWebSocket)

	// Text-only endpoint (for text-only clients)
	r.GET("/ws/text", rm.handleTextWebSocket)

	// Keep legacy demo endpoint temporarily for comparison
	r.GET("/ws-legacy", rm.handleLegacyWebSocket)
}

func (rm *RoutesManager) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		rm.deps.Logger.Errorf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Create unique IDs for this connection
	userID := uuid.New()
	sessionID := uuid.New()
	deviceID := uuid.New()

	rm.deps.Logger.Infof("ws connected - UserID: %s, SessionID: %s, DeviceID: %s", userID, sessionID, deviceID)

	// Create VSS context
	vssCtx, vssCancel := context.WithCancel(context.Background())

	// Create VSS instance
	vssConfig := vss.DefaultVSSConfig()
	voiceSystem := vss.NewVSS(userID, sessionID, vssConfig, rm.deps.Configs, rm.deps.Logger)

	// Create user connection state
	userConn := &UserConnection{
		UserID:      userID,
		SessionID:   sessionID,
		DeviceID:    deviceID,
		Conn:        conn,
		AudioBuffer: audioring.New(1024 * 1024), // 1MB audio buffer
		VoiceSystem: voiceSystem,
		VSSContext:  vssCtx,
		VSSCancel:   vssCancel,
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
		IsActive:    true,
	}

	// Register user connection
	rm.registerUserConnection(userID, userConn)
	defer rm.cleanupUserConnection(userID)

	// Start VSS in a separate goroutine
	go voiceSystem.Run(vssCtx)

	// Start VSS interrupt handler
	go rm.handleVSSInterrupts(userConn)

	// Set up device with full capabilities
	fullCaps := device.Capabilities{AudioSink: true, AudioWrite: true, TextSink: true}
	rm.deps.DeviceRegistry.UpsertDevice(userID, device.Device{
		UserID:    userID,
		SessionID: sessionID,
		Caps:      fullCaps,
		DeviceID:  deviceID,
	})

	// Create separate endpoints for text and audio
	textEndpoint := websockete.New(conn, device.Capabilities{TextSink: true})
	audioEndpoint := websockete.New(conn, device.Capabilities{AudioSink: true, AudioWrite: true})

	userConn.TextEndpoint = textEndpoint
	userConn.AudioEndpoint = audioEndpoint

	// Attach endpoints to registry
	rm.deps.DeviceRegistry.AttachEndpoint(userID, deviceID, textEndpoint)
	rm.deps.DeviceRegistry.AttachEndpoint(userID, deviceID, audioEndpoint)

	// Create brain system for this user
	userConn.BrainSystem = rm.createBrainSystem(userID, sessionID)

	// Handle incoming messages
	rm.handleConnectionMessages(userConn)
}

func (rm *RoutesManager) handleConnectionMessages(userConn *UserConnection) {
	for {
		messageType, msgBytes, err := userConn.Conn.ReadMessage()
		if err != nil {
			rm.deps.Logger.Errorf("ws read error for user %s: %v", userConn.UserID, err)
			break
		}

		userConn.LastActive = time.Now()

		switch messageType {
		case websocket.TextMessage:
			rm.handleTextMessage(userConn, msgBytes)
		case websocket.BinaryMessage:
			rm.handleBinaryMessage(userConn, msgBytes)
		default:
			rm.deps.Logger.Warnf("unknown message type %d from user %s", messageType, userConn.UserID)
		}
	}
}

func (rm *RoutesManager) handleTextMessage(userConn *UserConnection, msgBytes []byte) {
	var wsMsg WSMessage
	if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
		// Fallback: treat as plain text message
		rm.processTextInput(userConn, string(msgBytes))
		return
	}

	switch wsMsg.Type {
	case MessageTypeText:
		if textMsg, ok := wsMsg.Data.(map[string]interface{}); ok {
			if content, exists := textMsg["content"].(string); exists {
				rm.processTextInput(userConn, content)
			}
		}
	case MessageTypeInit:
		rm.handleInitMessage(userConn, wsMsg)
	default:
		rm.deps.Logger.Warnf("unhandled text message type %s from user %s", wsMsg.Type, userConn.UserID)
	}
}

func (rm *RoutesManager) handleBinaryMessage(userConn *UserConnection, msgBytes []byte) {
	// First 8 bytes: sample rate (4) + channels (2) + reserved (2)
	if len(msgBytes) < 8 {
		rm.deps.Logger.Errorf("invalid audio message size %d from user %s", len(msgBytes), userConn.UserID)
		return
	}

	// Extract audio metadata from first 8 bytes
	sampleRate := int32(msgBytes[0]) | int32(msgBytes[1])<<8 | int32(msgBytes[2])<<16 | int32(msgBytes[3])<<24
	channels := int16(msgBytes[4]) | int16(msgBytes[5])<<8

	// Rest is PCM data
	pcmData := msgBytes[8:]

	rm.processAudioInput(userConn, sampleRate, channels, pcmData)
}

func (rm *RoutesManager) processTextInput(userConn *UserConnection, text string) {
	msg := conversation.Message{
		Id:        uuid.New().String(),
		UserId:    userConn.UserID.String(),
		Text:      text,
		Timestamp: time.Now(),
		MsgRole:   assistant.USER,
		Tags:      []string{"websocket", "text"},
	}

	rm.deps.Logger.Infof("Processing text message for user %s: %s", userConn.UserID, text)

	// Process through brain system
	go func() {
		ctx := context.Background()
		err := userConn.BrainSystem.ProcessMessageWithStreaming(ctx, userConn.UserID, userConn.SessionID, msg)
		if err != nil {
			rm.deps.Logger.Errorf("brain processing error for user %s: %v", userConn.UserID, err)
		}
	}()
}

func (rm *RoutesManager) processAudioInput(userConn *UserConnection, sampleRate int32, channels int16, pcmData []byte) {
	// Rate limiting: track audio frames per second per user
	now := time.Now()
	if userConn.LastActive.Add(time.Second).Before(now) {
		// Reset rate limit counter every second
		userConn.LastActive = now
	}

	// Skip if we're receiving too much audio (basic flow control)
	audioInput := audioring.AudioInput{
		Data:       pcmData,
		Timestamp:  now,
		SampleRate: sampleRate,
		Channels:   channels,
	}

	// Store in audio buffer with graceful overflow handling
	if err := userConn.AudioBuffer.Enqueue(audioInput); err != nil {
		// Only log actual errors, not buffer overflow (which is handled gracefully)
		if err.Error() != "audio frame too large for buffer" {
			rm.deps.Logger.Debugf("Audio buffer issue for user %s: %v", userConn.UserID, err)
		} else {
			rm.deps.Logger.Warnf("Audio frame too large for user %s: %d bytes", userConn.UserID, len(pcmData))
		}
		return
	}

	// Log less frequently to reduce spam - only log larger frames (batched audio)
	if len(pcmData) > 8000 {
	}

	// Send audio to VSS with flow control
	if userConn.VoiceSystem != nil {
		audioEvent := vss.VSSEvent{
			Type:      vss.EventAudioInput,
			UserID:    userConn.UserID,
			SessionID: userConn.SessionID,
			Data: vss.AudioInputData{
				AudioInput: audioInput,
			},
			Timestamp: now,
		}

		select {
		case userConn.VoiceSystem.GetInputChannel() <- audioEvent:
			// Successfully sent to VSS
		default:
			rm.deps.Logger.Warnf("VSS input channel full for user %s, dropping audio frame", userConn.UserID)
		}
	}
}

func (rm *RoutesManager) handleInitMessage(userConn *UserConnection, wsMsg WSMessage) {
	if initData, ok := wsMsg.Data.(map[string]interface{}); ok {
		rm.deps.Logger.Infof("received init message from user %s: %+v", userConn.UserID, initData)
		// Handle capability updates or other initialization
	}
}

// handleAudioWebSocket handles audio-only WebSocket connections
func (rm *RoutesManager) handleAudioWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		rm.deps.Logger.Errorf("audio ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID := uuid.New()
	sessionID := uuid.New()
	deviceID := uuid.New()

	rm.deps.Logger.Infof("audio ws connected - UserID: %s", userID)

	// Create audio-only capabilities
	audioCaps := device.Capabilities{AudioSink: true, AudioWrite: true}

	// Register device and endpoint
	rm.deps.DeviceRegistry.UpsertDevice(userID, device.Device{
		UserID:    userID,
		SessionID: sessionID,
		Caps:      audioCaps,
		DeviceID:  deviceID,
	})

	audioEndpoint := websockete.New(conn, audioCaps)
	rm.deps.DeviceRegistry.AttachEndpoint(userID, deviceID, audioEndpoint)

	// Create audio buffer
	audioBuffer := audioring.New(500 * 1024) // 500KB for audio-only

	// Handle audio messages
	for {
		messageType, msgBytes, err := conn.ReadMessage()
		if err != nil {
			rm.deps.Logger.Errorf("audio ws read error: %v", err)
			break
		}

		if messageType == websocket.BinaryMessage {
			// Process audio data
			if len(msgBytes) >= 8 {
				sampleRate := int32(msgBytes[0]) | int32(msgBytes[1])<<8 | int32(msgBytes[2])<<16 | int32(msgBytes[3])<<24
				channels := int16(msgBytes[4]) | int16(msgBytes[5])<<8
				pcmData := msgBytes[8:]

				audioInput := audioring.AudioInput{
					Data:       pcmData,
					Timestamp:  time.Now(),
					SampleRate: sampleRate,
					Channels:   channels,
				}

				if err := audioBuffer.Enqueue(audioInput); err != nil {
					rm.deps.Logger.Errorf("failed to enqueue audio: %v", err)
				}

				rm.deps.Logger.Debugf("Audio frame: %d bytes, %dHz, %d channels", len(pcmData), sampleRate, channels)
			}
		}
	}
}

// handleTextWebSocket handles text-only WebSocket connections
func (rm *RoutesManager) handleTextWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		rm.deps.Logger.Errorf("text ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	userID := uuid.New()
	sessionID := uuid.New()
	deviceID := uuid.New()

	rm.deps.Logger.Infof("text ws connected - UserID: %s", userID)

	// Create text-only capabilities
	textCaps := device.Capabilities{TextSink: true}

	// Register device and endpoint
	rm.deps.DeviceRegistry.UpsertDevice(userID, device.Device{
		UserID:    userID,
		SessionID: sessionID,
		Caps:      textCaps,
		DeviceID:  deviceID,
	})

	textEndpoint := websockete.New(conn, textCaps)
	rm.deps.DeviceRegistry.AttachEndpoint(userID, deviceID, textEndpoint)

	// Create brain system for text processing
	brainSystem := rm.createBrainSystem(userID, sessionID)

	// Handle text messages
	for {
		messageType, msgBytes, err := conn.ReadMessage()
		if err != nil {
			rm.deps.Logger.Errorf("text ws read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			text := string(msgBytes)

			// Process as conversation message
			msg := conversation.Message{
				Id:        uuid.New().String(),
				UserId:    userID.String(),
				Text:      text,
				Timestamp: time.Now(),
				MsgRole:   assistant.USER,
				Tags:      []string{"websocket", "text-only"},
			}

			// Process through brain system
			go func() {
				ctx := context.Background()
				err := brainSystem.ProcessMessageWithStreaming(ctx, userID, sessionID, msg)
				if err != nil {
					rm.deps.Logger.Errorf("text brain processing error: %v", err)
				}
			}()
		}
	}
}

// registerUserConnection registers a new user connection
func (rm *RoutesManager) registerUserConnection(userID uuid.UUID, userConn *UserConnection) {
	rm.connectionMutex.Lock()
	defer rm.connectionMutex.Unlock()
	rm.userConnections[userID] = userConn
}

// cleanupUserConnection removes user connection when disconnected
func (rm *RoutesManager) cleanupUserConnection(userID uuid.UUID) {
	rm.connectionMutex.Lock()
	defer rm.connectionMutex.Unlock()

	if userConn, exists := rm.userConnections[userID]; exists {
		userConn.IsActive = false

		// Cancel VSS context to stop the voice system
		if userConn.VSSCancel != nil {
			userConn.VSSCancel()
		}

		duration := time.Since(userConn.ConnectedAt)
		rm.deps.Logger.Infof("cleaning up user connection for %s (connected for %v)", userID, duration)
		delete(rm.userConnections, userID)
	}
}

// handleVSSInterrupts processes interrupts from the Voice Streaming System
func (rm *RoutesManager) handleVSSInterrupts(userConn *UserConnection) {

	for {
		select {
		case <-userConn.VSSContext.Done():
			return

		case vssEvent := <-userConn.VoiceSystem.GetOutputChannel():
			rm.processVSSEvent(userConn, vssEvent)
		}
	}
}

// processVSSEvent handles events from the VSS
func (rm *RoutesManager) processVSSEvent(userConn *UserConnection, event vss.VSSEvent) {
	switch event.Type {
	case vss.EventInterrupt:
		rm.handleVSSInterrupt(userConn, event)
	case vss.EventListening:
		userConn.SendEvent(context.Background(), pipeline.GenericEvent{Key: string(event.Type), Value: event})
	default:
		rm.deps.Logger.Warnf("Unknown VSS event type %s from user %s", event.Type, userConn.UserID)
	}
}

// handleVSSInterrupt processes VSS interrupt events with transcription
func (rm *RoutesManager) handleVSSInterrupt(userConn *UserConnection, event vss.VSSEvent) {
	if interruptData, ok := event.Data.(vss.InterruptData); ok {
		rm.deps.Logger.Infof("VSS interrupt for user %s: %s (confidence: %.2f)",
			userConn.UserID, interruptData.Transcription, interruptData.Confidence)

		// Create conversation message from transcription
		msg := conversation.Message{
			Id:        uuid.New().String(),
			UserId:    userConn.UserID.String(),
			Text:      interruptData.Transcription,
			Timestamp: interruptData.StartTime,
			MsgRole:   assistant.USER,
			Tags:      []string{"vss", "voice", "transcribed"},
		}

		// Process through brain system
		go func() {
			ctx := context.Background()
			err := userConn.BrainSystem.ProcessMessageWithStreaming(ctx, userConn.UserID, userConn.SessionID, msg)
			if err != nil {
				rm.deps.Logger.Errorf("brain processing error for VSS interrupt from user %s: %v", userConn.UserID, err)
			} else {
				// Send AUD_PROC_DONE event back to VSS
				audProcDoneEvent := vss.VSSEvent{
					Type:      vss.EventAudProcDone,
					UserID:    userConn.UserID,
					SessionID: userConn.SessionID,
					Timestamp: time.Now(),
				}

				select {
				case userConn.VoiceSystem.GetInputChannel() <- audProcDoneEvent:
					// Successfully notified VSS
				default:
					rm.deps.Logger.Warnf("Could not send AUD_PROC_DONE to VSS for user %s", userConn.UserID)
				}
			}
		}()
	}
}

// createBrainSystem creates a new brain system for user
func (rm *RoutesManager) createBrainSystem(userID, sessionID uuid.UUID) *brain.BrainSystem {
	piperURL, _ := url.Parse("http://tts-piper:5000") // TODO: Make configurable
	brainSys := brain.NewBrainSystem(
		rm.deps.BrainConfig,
		rm.deps.Mux,
		rm.deps.DeviceRegistry,
		piperURL,
		*rm.deps.Logger,
	)

	rm.deps.Logger.Infof("created new brain system for user %s", userID)
	return brainSys
}

// getUserConnection gets user connection by userID
func (rm *RoutesManager) getUserConnection(userID uuid.UUID) (*UserConnection, bool) {
	rm.connectionMutex.RLock()
	defer rm.connectionMutex.RUnlock()

	conn, exists := rm.userConnections[userID]
	return conn, exists
}

// getAudioInputEndpoint gets the most recently used audio input endpoint for a user
func (rm *RoutesManager) getAudioInputEndpoint(userID uuid.UUID) (device.Endpoint, bool) {
	// For audio input, we need an endpoint that can receive audio (AudioWrite capability)
	audioWriteCaps := &device.Capabilities{AudioWrite: true}
	return rm.deps.DeviceRegistry.SelectEndpointWithMRU(userID, audioWriteCaps)
}

// getAudioSinkEndpoint gets the most recently used audio sink endpoint for a user
func (rm *RoutesManager) getAudioSinkEndpoint(userID uuid.UUID) (device.Endpoint, bool) {
	// For audio output, we need an endpoint that can sink audio
	audioSinkCaps := &device.Capabilities{AudioSink: true}
	return rm.deps.DeviceRegistry.SelectEndpointWithMRU(userID, audioSinkCaps)
}

// getTextSinkEndpoint gets the most recently used text sink endpoint for a user
func (rm *RoutesManager) getTextSinkEndpoint(userID uuid.UUID) (device.Endpoint, bool) {
	// For text output, we need an endpoint that can sink text
	textSinkCaps := &device.Capabilities{TextSink: true}
	return rm.deps.DeviceRegistry.SelectEndpointWithMRU(userID, textSinkCaps)
}

// Legacy WebSocket handler for comparison
func (rm *RoutesManager) handleLegacyWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	uid := uuid.New()
	sid := uuid.New()
	did := uuid.New()

	// Use local registry for legacy demo
	rg := memoryregistry.New()
	cps := device.Capabilities{AudioSink: true, TextSink: true}
	rg.UpsertDevice(uid, device.Device{
		UserID:    uid,
		SessionID: sid,
		Caps:      cps,
		DeviceID:  did,
	})

	rg.AttachEndpoint(uid, did, websockete.New(conn, cps))

	// Parse the URL string to url.URL type
	ollamaURL, err := url.Parse("http://traefik/v1/llm/ollama")
	if err != nil {
		log.Fatalf("invalid ollama URL: %v", err)
	}
	llamaPv := olp.New(config.OllamaConfig{
		LLamaModels: []config.LLMModelConfig{
			{Name: "llama3:8b", Url: *ollamaURL},
		},
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}

		// Run demo pipeline when a msg arrives
		in := adapters.ContractInput{
			Msgs: []adapters.ContractMessage{
				{
					Content:   string(msg),
					Role:      adapters.USER,
					CreatedAt: time.Now(),
				},
			},
		}
		go QuickDemo(context.Background(), rg, &llamaPv, in, uid, sid)
	}
}

func QuickDemo(
	ctx context.Context, rg registry.DeviceRegistry, llamaPv *olp.OllamaProvider, in adapters.ContractInput,
	userID uuid.UUID, sessionID uuid.UUID,
) {
	// setup LLM adapter
	// Batch LLM deltas to avoid word-by-word streaming
	llamaAd := ollama.New(llamaPv, adapters.ContractLLMCfg{
		DeltaTimeDuration: 200 * time.Millisecond,
		DeltaBufferLimit:  32,
	}, nil)
	ads := []adapters.ContractAdapter{llamaAd}
	mux := router.New(ads)

	// setup publisher and TTS
	pub := io.New(rg)
	// Talk directly to Piper HTTP inside the Docker network
	// rhasspy/wyoming-piper exposes HTTP on 5000 mapped to 5003 externally
	piperClient := piper.New("http://tts-piper:5000")
	str := stream.New(&piperClient)
	pl := pipeline.New(&str, &pub)

	// wire it
	rsp := make(adapters.ContractResponseChannel)
	go func() {
		err := mux.Stream(ctx, in, &rsp)
		if err != nil {
			log.Printf("got error %v", err)
		}
	}()
	_ = pl.Broadcast(ctx, userID, sessionID, &rsp)

	log.Printf("demo finished for user %s / session %s", userID, sessionID)
}
