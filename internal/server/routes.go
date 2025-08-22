package server

import (
	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
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
}
