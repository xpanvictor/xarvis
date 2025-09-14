package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	vss "github.com/xpanvictor/xarvis/internal/domains/sys_manager/voice_stream_system"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/io/device"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
)

// WebSocketHandler handles WebSocket connections and routes
type WebSocketHandler struct {
	logger              *Logger.Logger
	conversationService conversation.ConversationService
	deviceRegistry      registry.DeviceRegistry
	config              *config.Settings
	connectionManager   *ConnectionManager
	upgrader            websocket.Upgrader
	userService         user.UserService // Add user service for token validation
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	logger *Logger.Logger,
	conversationService conversation.ConversationService,
	deviceRegistry registry.DeviceRegistry,
	config *config.Settings,
	userService user.UserService,
) *WebSocketHandler {
	return &WebSocketHandler{
		logger:              logger,
		conversationService: conversationService,
		deviceRegistry:      deviceRegistry,
		config:              config,
		connectionManager:   NewConnectionManager(logger),
		userService:         userService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking for production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(router gin.IRouter) {
	ws := router.Group("/ws")
	{
		ws.GET("/", h.HandleMainWebSocket)
		ws.GET("/audio", h.HandleAudioWebSocket)
		ws.GET("/text", h.HandleTextWebSocket)
		ws.GET("/stats", h.HandleStats)
	}
}

// HandleMainWebSocket handles the main WebSocket connection with full capabilities
func (h *WebSocketHandler) HandleMainWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Extract user ID from authenticated token
	token := c.Query("token")
	userIDStr := c.Query("userId")

	h.logger.Infof("ws token: %v", token)
	h.logger.Infof("ws id users id is %v", userIDStr)
	h.logger.Infof("Full request URL: %v", c.Request.URL.String())
	h.logger.Infof("Query params: %v", c.Request.URL.Query())

	var userID uuid.UUID

	// If token is provided, validate it and get the real user ID
	if token != "" {
		claims, err := h.userService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			h.logger.Errorf("WebSocket token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Parse the userID from claims
		parsedUserID, err := uuid.Parse(claims.UserID)
		if err != nil {
			h.logger.Errorf("Invalid userID in token claims: %s", claims.UserID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		userID = parsedUserID
		h.logger.Infof("Authenticated WebSocket user: %s", userID)
	} else if userIDStr != "" {
		// Fallback to query parameter (for testing/development)
		if parsedID, err := uuid.Parse(userIDStr); err == nil {
			userID = parsedID
			h.logger.Warnf("Using unauthenticated userId from query parameter: %s", userID)
		} else {
			h.logger.Warnf("Invalid userId format received: '%s', generating new UUID", userIDStr)
			userID = uuid.New()
		}
	} else {
		h.logger.Debugf("No userId provided, generating new UUID")
		userID = uuid.New()
	}

	// Create session with full capabilities
	session := NewSession(userID, conn, device.Capabilities{
		AudioSink:  true,
		AudioWrite: true,
		TextSink:   true,
	})

	// Register connection and setup cleanup
	h.connectionManager.RegisterConnection(session)
	defer h.connectionManager.UnregisterConnection(session.UserID)

	// Register device and endpoint with device registry
	deviceEntry := device.Device{
		UserID:     userID,
		DeviceID:   session.DeviceID,
		SessionID:  session.SessionID,
		Caps:       session.Capabilities,
		LastActive: time.Now(),
		Endpoints:  make(map[device.EndpointID]device.Endpoint),
	}

	// Register device
	if err := h.deviceRegistry.UpsertDevice(userID, deviceEntry); err != nil {
		h.logger.Errorf("Failed to register device: %v", err)
		return
	}

	// Register endpoint
	if err := h.deviceRegistry.AttachEndpoint(userID, session.DeviceID, session); err != nil {
		h.logger.Errorf("Failed to register endpoint: %v", err)
		return
	}

	// Cleanup device registry on disconnect
	defer func() {
		if err := h.deviceRegistry.RemoveDevice(userID, session.DeviceID); err != nil {
			h.logger.Errorf("Failed to remove device: %v", err)
		}
	}()

	// Start VSS and message handling
	h.startVSSProcessing(session)
	h.handleConnection(session)
}

// HandleAudioWebSocket handles audio-only WebSocket connections
func (h *WebSocketHandler) HandleAudioWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Audio WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Extract user ID
	userIDStr := c.Query("userId")
	var userID uuid.UUID
	if userIDStr != "" {
		if parsedID, err := uuid.Parse(userIDStr); err == nil {
			userID = parsedID
		} else {
			h.logger.Warnf("Invalid userId format received in HandleAudioWebSocket: '%s', generating new UUID", userIDStr)
			userID = uuid.New()
		}
	} else {
		h.logger.Debugf("No userId provided in HandleAudioWebSocket, generating new UUID")
		userID = uuid.New()
	}

	// Create session with audio capabilities only
	session := NewSession(userID, conn, device.Capabilities{
		AudioSink:  true,
		AudioWrite: true,
		TextSink:   false,
	})

	h.connectionManager.RegisterConnection(session)
	defer h.connectionManager.UnregisterConnection(session.UserID)

	// Register device and endpoint with device registry
	deviceEntry := device.Device{
		UserID:     userID,
		DeviceID:   session.DeviceID,
		SessionID:  session.SessionID,
		Caps:       session.Capabilities,
		LastActive: time.Now(),
		Endpoints:  make(map[device.EndpointID]device.Endpoint),
	}

	// Register device
	if err := h.deviceRegistry.UpsertDevice(userID, deviceEntry); err != nil {
		h.logger.Errorf("Failed to register audio device: %v", err)
		return
	}

	// Register endpoint
	if err := h.deviceRegistry.AttachEndpoint(userID, session.DeviceID, session); err != nil {
		h.logger.Errorf("Failed to register audio endpoint: %v", err)
		return
	}

	// Cleanup device registry on disconnect
	defer func() {
		if err := h.deviceRegistry.RemoveDevice(userID, session.DeviceID); err != nil {
			h.logger.Errorf("Failed to remove audio device: %v", err)
		}
	}()

	h.startVSSProcessing(session)
	h.handleConnection(session)
}

// HandleTextWebSocket handles text-only WebSocket connections
func (h *WebSocketHandler) HandleTextWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Text WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Extract user ID
	userIDStr := c.Query("userId")
	var userID uuid.UUID
	if userIDStr != "" {
		if parsedID, err := uuid.Parse(userIDStr); err == nil {
			userID = parsedID
		} else {
			h.logger.Warnf("Invalid userId format received in HandleTextWebSocket: '%s', generating new UUID", userIDStr)
			userID = uuid.New()
		}
	} else {
		h.logger.Debugf("No userId provided in HandleTextWebSocket, generating new UUID")
		userID = uuid.New()
	}

	// Create session with text capabilities only
	session := NewSession(userID, conn, device.Capabilities{
		AudioSink:  false,
		AudioWrite: false,
		TextSink:   true,
	})

	h.connectionManager.RegisterConnection(session)
	defer h.connectionManager.UnregisterConnection(session.UserID)

	// Register device and endpoint with device registry
	deviceEntry := device.Device{
		UserID:     userID,
		DeviceID:   session.DeviceID,
		SessionID:  session.SessionID,
		Caps:       session.Capabilities,
		LastActive: time.Now(),
		Endpoints:  make(map[device.EndpointID]device.Endpoint),
	}

	// Register device
	if err := h.deviceRegistry.UpsertDevice(userID, deviceEntry); err != nil {
		h.logger.Errorf("Failed to register text device: %v", err)
		return
	}

	// Register endpoint
	if err := h.deviceRegistry.AttachEndpoint(userID, session.DeviceID, session); err != nil {
		h.logger.Errorf("Failed to register text endpoint: %v", err)
		return
	}

	// Cleanup device registry on disconnect
	defer func() {
		if err := h.deviceRegistry.RemoveDevice(userID, session.DeviceID); err != nil {
			h.logger.Errorf("Failed to remove text device: %v", err)
		}
	}()

	// No VSS for text-only connections
	h.handleConnection(session)
}

// HandleStats provides connection statistics
func (h *WebSocketHandler) HandleStats(c *gin.Context) {
	stats := h.connectionManager.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   stats,
	})
}

// startVSSProcessing initializes and starts VSS for a session
func (h *WebSocketHandler) startVSSProcessing(session *Session) {
	// Skip VSS for text-only sessions
	if !session.Capabilities.AudioSink && !session.Capabilities.AudioWrite {
		h.logger.Infof("Skipping VSS for text-only session %s", session.SessionID)
		return
	}

	// Create VSS
	vssConfig := vss.DefaultVSSConfig()
	vssCtx, vssCancel := context.WithCancel(context.Background())

	session.VSSContext = vssCtx
	session.VSSCancel = vssCancel
	session.VoiceSystem = vss.NewVSS(
		session.UserID,
		session.SessionID,
		vssConfig,
		h.config,
		h.logger,
	)

	// Start VSS
	go session.VoiceSystem.Run(vssCtx)

	// Start VSS event bridge
	inputManager := NewInputStreamManager(h.logger, h.conversationService)
	bridge := NewVSSEventBridge(h.logger, inputManager, h.connectionManager)
	go bridge.HandleVSSEvents(vssCtx, session)

	h.logger.Infof("Started VSS processing for session %s", session.SessionID)
}

// handleConnection handles the main WebSocket connection loop
func (h *WebSocketHandler) handleConnection(session *Session) {
	inputManager := NewInputStreamManager(h.logger, h.conversationService)

	h.logger.Infof("Starting WebSocket connection handling for session %s", session.SessionID)

	for {
		messageType, data, err := session.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Errorf("WebSocket read error: %v", err)
			} else {
				h.logger.Infof("WebSocket connection closed for session %s", session.SessionID)
			}
			break
		}

		// Update last activity
		session.UpdateLastActive()

		switch messageType {
		case websocket.TextMessage:
			h.handleTextMessage(session, data, inputManager)
		case websocket.BinaryMessage:
			h.handleBinaryMessage(session, data, inputManager)
		}
	}
}

// handleTextMessage processes incoming text messages
func (h *WebSocketHandler) handleTextMessage(session *Session, data []byte, inputManager *InputStreamManager) {
	var wsMsg WSMessage
	if err := json.Unmarshal(data, &wsMsg); err != nil {
		h.logger.Errorf("Failed to unmarshal WebSocket message: %v", err)
		session.SendError("INVALID_MESSAGE", "Invalid message format")
		return
	}

	ctx := context.Background()

	switch wsMsg.Type {
	case MessageTypeInit:
		h.handleInitMessage(session, wsMsg)

	case MessageTypeText:
		if textMsg, ok := wsMsg.Data.(map[string]interface{}); ok {
			if content, exists := textMsg["content"].(string); exists {
				err := inputManager.HandleTextInput(ctx, session, content)
				if err != nil {
					h.logger.Errorf("Failed to handle text input: %v", err)
					session.SendError("TEXT_PROCESSING_ERROR", "Failed to process text input")
				}
			}
		}

	case MessageTypeListeningControl:
		if controlData, ok := wsMsg.Data.(map[string]interface{}); ok {
			if action, exists := controlData["action"].(string); exists {
				control := ListeningControl{Action: action}
				err := inputManager.HandleListeningControl(ctx, session, control)
				if err != nil {
					h.logger.Errorf("Failed to handle listening control: %v", err)
					session.SendError("CONTROL_ERROR", "Failed to process listening control")
				}
			}
		}

	case MessageTypeAudio:
		// Handle audio data sent as JSON (base64 encoded)
		if _, ok := wsMsg.Data.(map[string]interface{}); ok {
			// TODO: Implement audio data extraction and processing
			h.logger.Debugf("Received audio data in text message for session %s", session.SessionID)
		}

	default:
		h.logger.Warnf("Unknown message type: %s", wsMsg.Type)
		session.SendError("UNKNOWN_MESSAGE_TYPE", fmt.Sprintf("Unknown message type: %s", wsMsg.Type))
	}
}

// handleBinaryMessage processes incoming binary messages (typically audio)
func (h *WebSocketHandler) handleBinaryMessage(session *Session, data []byte, inputManager *InputStreamManager) {
	// For now, treat binary data as raw audio (PCM)
	// TODO: Implement proper audio format detection and parsing
	audioMsg := AudioMessage{
		SampleRate: 16000, // Default sample rate
		Channels:   1,     // Mono
		Data:       data,
	}

	ctx := context.Background()
	err := inputManager.HandleAudioInput(ctx, session, audioMsg)
	if err != nil {
		h.logger.Errorf("Failed to handle audio input: %v", err)
		session.SendError("AUDIO_PROCESSING_ERROR", "Failed to process audio input")
	}
}

// handleInitMessage processes initialization messages
func (h *WebSocketHandler) handleInitMessage(session *Session, wsMsg WSMessage) {
	if initData, ok := wsMsg.Data.(map[string]interface{}); ok {
		h.logger.Infof("Received init message for session %s: %v", session.SessionID, initData)

		// Send acknowledgment
		session.SendWebSocketMessage(MessageTypeInit, map[string]interface{}{
			"status":    "connected",
			"sessionId": session.SessionID.String(),
			"userId":    session.UserID.String(),
		})
	}
}

// Close shuts down the WebSocket handler
func (h *WebSocketHandler) Close() error {
	return h.connectionManager.Close()
}
