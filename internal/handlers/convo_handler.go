package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// SearchMemories searches for memories based on query and optional filters
// @Summary Search user memories
// @Description Searches for memories belonging to the authenticated user based on query and optional filters
// @Tags Conversation
// @Accept json
// @Produce json
// @Param request body MemorySearchRequest true "Memory search parameters"
// @Success 200 {object} MemoriesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /conversation/memories/search [post]
func (h *ConversationHandler) SearchMemories(c *gin.Context) {
	UserInfo, ok := ExtractUserInfo(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var searchReq MemorySearchRequest
	if err := c.ShouldBindJSON(&searchReq); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	// Set default limit if not specified
	if searchReq.Limit <= 0 {
		searchReq.Limit = 50 // Default to 50 memories
	}

	memories, err := h.convoService.SearchMemories(c, UserInfo.UserID, searchReq.Query, searchReq.Type, searchReq.Limit)
	if err != nil {
		h.logger.Errorf("search memories error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, MemoriesResponse{
		Memories: memories,
		Count:    len(memories),
	})
}

// DeleteMemory deletes a specific memory by ID
// @Summary Delete a memory
// @Description Deletes a specific memory belonging to the authenticated user
// @Tags Conversation
// @Param id path string true "Memory ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /conversation/memories/{id} [delete]
func (h *ConversationHandler) DeleteMemory(c *gin.Context) {
	UserInfo, ok := ExtractUserInfo(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	memoryIDStr := c.Param("id")
	memoryID, err := uuid.Parse(memoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid memory ID format",
			Details: err.Error(),
		})
		return
	}

	err = h.convoService.DeleteMemory(c, UserInfo.UserID, memoryID)
	if err != nil {
		h.logger.Errorf("delete memory error: %v", err)
		if err.Error() == "memory with ID "+memoryIDStr+" not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Memory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterConversationRoutes registers all conversation-related routes
func (h *ConversationHandler) RegisterConversationRoutes(r *gin.RouterGroup, userService user.UserService) {
	// Protected routes (authentication required)
	protected := r.Group("/conversation")
	protected.Use(AuthMiddleware(userService, h.logger))
	{
		protected.POST("/message", h.ProcessMessage)
		protected.POST("/memory", h.CreateMemory)
		protected.POST("/memories/search", h.SearchMemories)
		protected.DELETE("/memories/:id", h.DeleteMemory)
		protected.GET("", h.RetrieveConversation)
	}
}
