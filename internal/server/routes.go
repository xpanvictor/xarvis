package server

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	olp "github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io"
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
	return Dependencies{
		conversationRepository: conversationRepository,
	}
}

func InitializeRoutes(r *gin.Engine, dep Dependencies) {
	// setup services & handlers
	r.GET("/", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"message": "Server healthy"}) })
	r.GET("/health", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"status": "ok"}) })
	// quick ws test
}

func QuickDemo(
	ctx context.Context, cfg config.OllamaConfig, in adapters.ContractInput,
	userID uuid.UUID, sessionID uuid.UUID,
) {
	llamaPv := olp.New(cfg)
	llamaAd := ollama.New(llamaPv, adapters.ContractLLMCfg{}, nil)
	ads := []adapters.ContractAdapter{llamaAd}
	mux := router.New(ads)
	rg := memoryregistry.New()
	pub := io.New(rg)
	piperClient := piper.New("localhost:80/v1/tts")
	str := stream.New(&piperClient)
	pipeline := pipeline.New(&str, &pub)

	rsp := make(adapters.ContractResponseChannel)
	mux.Stream(ctx, in, &rsp)
	pipeline.Broadcast(ctx, userID, sessionID, &rsp)
}
