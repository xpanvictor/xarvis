package server

import (
	"context"
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
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper/stream"
)

type Dependencies struct {
	conversationRepository conversation.ConversationRepository
	// New brain system dependencies
	DeviceRegistry registry.DeviceRegistry
	Mux            *router.Mux
	BrainConfig    config.BrainConfig
	Logger         *Logger.Logger
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
	deps         Dependencies
	userSystems  map[uuid.UUID]*UserBrainSystem // Track active brain systems per user
	systemsMutex sync.RWMutex
}

func NewServerDependencies(
	conversationRepository conversation.ConversationRepository,
	deviceRegistry registry.DeviceRegistry,
	mux *router.Mux,
	brainConfig config.BrainConfig,
	logger *Logger.Logger,
) Dependencies {
	return Dependencies{
		conversationRepository: conversationRepository,
		DeviceRegistry:         deviceRegistry,
		Mux:                    mux,
		BrainConfig:            brainConfig,
		Logger:                 logger,
	}
}

func NewRoutesManager(deps Dependencies) *RoutesManager {
	return &RoutesManager{
		deps:        deps,
		userSystems: make(map[uuid.UUID]*UserBrainSystem),
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

	// New WebSocket endpoint using brain system
	r.GET("/ws", rm.handleWebSocket)

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
	userID := uuid.New() // Each device gets unique userID for isolation
	sessionID := uuid.New()
	connectionID := uuid.New() // Track this specific connection

	rm.deps.Logger.Infof("ws connected - UserID: %s, SessionID: %s, ConnectionID: %s", userID, sessionID, connectionID)

	// Get or create brain system for this user
	brainSys := rm.getOrCreateBrainSystem(userID, sessionID)
	rm.deps.Logger.Infof("Brain system assigned for UserID: %s", userID)

	// Set up device in registry with capabilities
	deviceID := uuid.New()
	caps := device.Capabilities{AudioSink: true, TextSink: true}

	rm.deps.Logger.Infof("Registering device - UserID: %s, DeviceID: %s", userID, deviceID)

	rm.deps.DeviceRegistry.UpsertDevice(userID, device.Device{
		UserID:    userID,
		SessionID: sessionID,
		Caps:      caps,
		DeviceID:  deviceID,
	})

	// Attach WebSocket endpoint
	rm.deps.DeviceRegistry.AttachEndpoint(userID, deviceID, websockete.New(conn, caps))
	rm.deps.Logger.Infof("WebSocket endpoint attached - UserID: %s, DeviceID: %s", userID, deviceID)

	// Handle incoming messages
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			rm.deps.Logger.Errorf("ws read error: %v", err)
			rm.cleanupUserSystem(userID)
			break
		}

		rm.deps.Logger.Infof("ws got message from UserID %s (ConnectionID: %s): %s", userID, connectionID, msgBytes)

		// Create conversation message
		msg := conversation.Message{
			Id:        uuid.New().String(),
			UserId:    userID.String(),
			Text:      string(msgBytes),
			Timestamp: time.Now(),
			MsgRole:   assistant.USER,
			Tags:      []string{"websocket", connectionID.String()},
		}

		rm.deps.Logger.Infof("Processing message for UserID: %s, SessionID: %s", userID, sessionID)

		// Process message through brain system with streaming
		go func() {
			ctx := context.Background()
			start := time.Now()
			rm.deps.Logger.Infof("Starting brain processing for user %s at %v", userID, start)
			err := brainSys.ProcessMessageWithStreaming(ctx, userID, sessionID, msg)
			duration := time.Since(start)
			if err != nil {
				rm.deps.Logger.Errorf("brain processing error after %v: %v", duration, err)
			} else {
				rm.deps.Logger.Infof("Brain processing completed for user %s after %v", userID, duration)
			}
		}()
	}
}

// getOrCreateBrainSystem gets existing brain system or creates new one for user
func (rm *RoutesManager) getOrCreateBrainSystem(userID, sessionID uuid.UUID) *brain.BrainSystem {
	rm.systemsMutex.Lock()
	defer rm.systemsMutex.Unlock()

	// Check if user already has an active brain system
	if userSys, exists := rm.userSystems[userID]; exists {
		// Update session ID for existing system
		userSys.sessionID = sessionID
		return userSys.brainSystem
	}

	// Create new brain system for user
	piperURL, _ := url.Parse("http://tts-piper:5000") // TODO: Make configurable
	brainSys := brain.NewBrainSystem(
		rm.deps.BrainConfig,
		rm.deps.Mux,
		rm.deps.DeviceRegistry,
		piperURL,
		*rm.deps.Logger,
	)

	rm.userSystems[userID] = &UserBrainSystem{
		brainSystem: brainSys,
		userID:      userID,
		sessionID:   sessionID,
		connectedAt: time.Now(),
	}

	rm.deps.Logger.Infof("created new brain system for user %s", userID)
	return brainSys
}

// cleanupUserSystem removes brain system when user disconnects
func (rm *RoutesManager) cleanupUserSystem(userID uuid.UUID) {
	rm.systemsMutex.Lock()
	defer rm.systemsMutex.Unlock()

	if userSys, exists := rm.userSystems[userID]; exists {
		rm.deps.Logger.Infof("cleaning up brain system for user %s (connected for %v)",
			userID, time.Since(userSys.connectedAt))
		delete(rm.userSystems, userID)
	}
}

// Legacy WebSocket handler for comparison
func (rm *RoutesManager) handleLegacyWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	log.Printf("ws connected")
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
		log.Printf("ws got: %s", msg)

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
