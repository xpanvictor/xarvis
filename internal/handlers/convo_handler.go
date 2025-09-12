package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/constants/prompts"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

type ConversationHandler struct {
	convoService conversation.ConversationService
	logger       *Logger.Logger
}

func NewConvoHandler(
	convoService conversation.ConversationService,
	logger *Logger.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		convoService: convoService,
		logger:       logger,
	}
}

// ProcessMessage assistant generates AI response
// @Summary Process user message and generate AI response
// @Description Processes a user message through the conversation service and returns an AI-generated response
// @Tags Conversation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body types.CreateMessage true "User message data"
// @Success 201 {object} MessageResponse "AI response message"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error or couldn't process message"
// @Router /conversation/message [post]
func (h *ConversationHandler) ProcessMessage(c *gin.Context) {
	UserInfo, ok := ExtractUserInfo(c)
	if !ok {
		return
	}

	var usrMsg types.CreateMessage
	if err := c.ShouldBindJSON(&usrMsg); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	resp, err := h.convoService.ProcessMsg(
		c,
		UserInfo.UserID,
		usrMsg.ToMessage(UserInfo.UserID),
		[]types.Message{prompts.DEFAULT_PROMPT.GetCurrentPrompt().ToMessage()},
	)
	if err != nil {
		switch err {
		case conversation.ErrProcMsg:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "couldn't process message, try later!"})
		default:
			h.logger.Errorf("process msg error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, MessageResponse{
		Message: *resp,
	})
}

// RetrieveConversation gets user's conversation history
// @Summary Retrieve user conversation
// @Description Retrieves the conversation history for the authenticated user including messages and memories
// @Tags Conversation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ConversationResponse "User conversation data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /conversation [get]
func (h *ConversationHandler) RetrieveConversation(c *gin.Context) {
	UserInfo, ok := ExtractUserInfo(c)
	if !ok {
		return
	}

	conversation, err := h.convoService.RetrieveConversation(c, UserInfo.UserID)
	if err != nil {
		h.logger.Errorf("retrieve conversation error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ConversationResponse{
		Conversation: *conversation,
	})
}

// CreateMemory creates a new memory for the user
// @Summary Create a new memory
// @Description Creates a new memory entry for the authenticated user's conversation
// @Tags Conversation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body types.CreateMemory true "Memory creation data"
// @Success 201 {object} MemoryResponse "Created memory"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /conversation/memory [post]
func (h *ConversationHandler) CreateMemory(c *gin.Context) {
	UserInfo, ok := ExtractUserInfo(c)
	if !ok {
		return
	}

	var createMemory types.CreateMemory
	if err := c.ShouldBindJSON(&createMemory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	memory := createMemory.ToMemory()
	createdMemory, err := h.convoService.CreateMemory(c, UserInfo.UserID, memory)
	if err != nil {
		h.logger.Errorf("create memory error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, MemoryResponse{
		Memory: *createdMemory,
	})
}

// RegisterConversationRoutes registers all conversation-related routes
func (h *ConversationHandler) RegisterConversationRoutes(r *gin.RouterGroup, userService user.UserService) {
	// Protected routes (authentication required)
	protected := r.Group("/conversation")
	protected.Use(AuthMiddleware(userService, h.logger))
	{
		protected.POST("/message", h.ProcessMessage)
		protected.POST("/memory", h.CreateMemory)
		protected.GET("", h.RetrieveConversation)
	}
}
