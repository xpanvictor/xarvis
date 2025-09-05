package server

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
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
}

func NewServerDependencies(
	conversationRepository conversation.ConversationRepository,
) Dependencies {
	return Dependencies{conversationRepository: conversationRepository}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev-only
}

func InitializeRoutes(cfg *config.Settings, r *gin.Engine, dep Dependencies) {
	r.GET("/", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"message": "Server healthy"}) })
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"status": "ok"}) })

	// --- Quick WebSocket demo ---
	rg := memoryregistry.New()
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
	// also client attaches another ws connection for input streaming
	r.GET("/ws", func(c *gin.Context) {
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

		cps := device.Capabilities{AudioSink: true, TextSink: true}
		rg.UpsertDevice(uid, device.Device{
			UserID:    uid,
			SessionID: sid,
			Caps:      cps,
			DeviceID:  did,
		})

		rg.AttachEndpoint(uid, did, websockete.New(conn, cps))

		// run this in own thread

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
	})
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
