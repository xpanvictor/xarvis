package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/note"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// NoteHandler handles note-related HTTP requests
type NoteHandler struct {
	noteService note.NoteService
	logger      *Logger.Logger
}

// NewNoteHandler creates a new note handler
func NewNoteHandler(noteService note.NoteService, logger *Logger.Logger) *NoteHandler {
	return &NoteHandler{
		noteService: noteService,
		logger:      logger,
	}
}

// CreateNote handles note creation
// @Summary Create a new note
// @Description Create a new note for the authenticated user
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body note.CreateNoteRequest true "Note creation data"
// @Success 201 {object} CreateNoteResponse "Note created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes [post]
func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req note.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	noteResponse, err := h.noteService.CreateNote(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case note.ErrInvalidNoteData:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid note data"})
		default:
			h.logger.Errorf("create note error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, CreateNoteResponse{
		Message: "Note created successfully",
		Note:    *noteResponse,
	})
}

// GetNote handles getting a specific note
// @Summary Get note by ID
// @Description Get a specific note by ID (user can only access their own notes)
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID"
// @Success 200 {object} NoteResponse "Note data"
// @Failure 400 {object} ErrorResponse "Invalid note ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Note not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes/{id} [get]
func (h *NoteHandler) GetNote(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Note ID is required"})
		return
	}

	noteResponse, err := h.noteService.GetNote(c.Request.Context(), userID, noteID)
	if err != nil {
		switch err {
		case note.ErrNoteNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Note not found"})
		case note.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("get note error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, NoteResponse{
		Note: *noteResponse,
	})
}

// UpdateNote handles updating a note
// @Summary Update note
// @Description Update a specific note (user can only update their own notes)
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID"
// @Param request body note.UpdateNoteRequest true "Note update data"
// @Success 200 {object} UpdateNoteResponse "Note updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Note not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes/{id} [put]
func (h *NoteHandler) UpdateNote(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Note ID is required"})
		return
	}

	var req note.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	noteResponse, err := h.noteService.UpdateNote(c.Request.Context(), userID, noteID, req)
	if err != nil {
		switch err {
		case note.ErrNoteNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Note not found"})
		case note.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		case note.ErrInvalidNoteData:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid note data"})
		default:
			h.logger.Errorf("update note error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateNoteResponse{
		Message: "Note updated successfully",
		Note:    *noteResponse,
	})
}

// DeleteNote handles note deletion
// @Summary Delete note
// @Description Delete a specific note (user can only delete their own notes)
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID"
// @Success 200 {object} SuccessResponse "Note deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid note ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Note not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes/{id} [delete]
func (h *NoteHandler) DeleteNote(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Note ID is required"})
		return
	}

	err := h.noteService.DeleteNote(c.Request.Context(), userID, noteID)
	if err != nil {
		switch err {
		case note.ErrNoteNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Note not found"})
		case note.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("delete note error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Note deleted successfully",
	})
}

// ListUserNotes handles listing user's notes with filtering
// @Summary List user notes
// @Description List all notes for the authenticated user with optional filtering
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search in note content"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param orderBy query string false "Order by field (created_at, content)" default(created_at)
// @Param order query string false "Order direction (asc, desc)" default(desc)
// @Param offset query int false "Number of notes to skip" default(0)
// @Param limit query int false "Number of notes to return" default(20)
// @Success 200 {object} ListNotesResponse "List of notes with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes [get]
func (h *NoteHandler) ListUserNotes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	// Parse query parameters
	var filters note.ListNotesRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Parse tags from comma-separated string if needed
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		filters.Tags = strings.Split(tagsStr, ",")
		// Trim whitespace from each tag
		for i, tag := range filters.Tags {
			filters.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Set default limits
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	notes, total, err := h.noteService.ListUserNotes(c.Request.Context(), userID, filters)
	if err != nil {
		h.logger.Errorf("list user notes error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListNotesResponse{
		Notes: notes,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// SearchNotes handles searching notes
// @Summary Search notes
// @Description Search notes by content and/or tags for the authenticated user
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param offset query int false "Number of notes to skip" default(0)
// @Param limit query int false "Number of notes to return" default(20)
// @Success 200 {object} SearchNotesResponse "Search results with pagination"
// @Failure 400 {object} ErrorResponse "Search query is required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes/search [get]
func (h *NoteHandler) SearchNotes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Search query is required"})
		return
	}

	// Parse pagination parameters
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// Parse tags
	var tags []string
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		// Trim whitespace from each tag
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	notes, total, err := h.noteService.SearchNotes(c.Request.Context(), userID, query, tags, offset, limit)
	if err != nil {
		h.logger.Errorf("search notes error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, SearchNotesResponse{
		Notes: notes,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
		Query: query,
		Tags:  tags,
	})
}

// GetNotesByTags handles getting notes by tags
// @Summary Get notes by tags
// @Description Get notes filtered by specific tags for the authenticated user
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tags query string true "Tags to filter by (comma-separated)"
// @Param offset query int false "Number of notes to skip" default(0)
// @Param limit query int false "Number of notes to return" default(20)
// @Success 200 {object} ListNotesResponse "List of notes with pagination"
// @Failure 400 {object} ErrorResponse "Tags parameter is required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /notes/tags [get]
func (h *NoteHandler) GetNotesByTags(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	tagsStr := c.Query("tags")
	if tagsStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Tags parameter is required"})
		return
	}

	// Parse tags from comma-separated string
	tags := strings.Split(tagsStr, ",")
	// Trim whitespace from each tag
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	// Parse pagination parameters
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	notes, total, err := h.noteService.GetNotesByTags(c.Request.Context(), userID, tags, offset, limit)
	if err != nil {
		h.logger.Errorf("get notes by tags error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListNotesResponse{
		Notes: notes,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	})
}

// ListAllNotes handles listing all notes (admin endpoint)
// @Summary List all notes (Admin)
// @Description List all notes in the system with optional filtering (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search in note content"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param orderBy query string false "Order by field (created_at, content)" default(created_at)
// @Param order query string false "Order direction (asc, desc)" default(desc)
// @Param offset query int false "Number of notes to skip" default(0)
// @Param limit query int false "Number of notes to return" default(20)
// @Success 200 {object} ListNotesResponse "List of notes with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/notes [get]
func (h *NoteHandler) ListAllNotes(c *gin.Context) {
	// Parse query parameters
	var filters note.ListNotesRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Parse tags from comma-separated string if needed
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		filters.Tags = strings.Split(tagsStr, ",")
		// Trim whitespace from each tag
		for i, tag := range filters.Tags {
			filters.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Set default limits
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	notes, total, err := h.noteService.ListAllNotes(c.Request.Context(), filters)
	if err != nil {
		h.logger.Errorf("list all notes error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListNotesResponse{
		Notes: notes,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// GetProjectNotes handles getting notes for a specific project (project logs)
// @Summary Get project notes/logs
// @Description Get all notes/logs for a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param offset query int false "Number of notes to skip" default(0)
// @Param limit query int false "Number of notes to return" default(20)
// @Success 200 {object} ListNotesResponse "List of project notes/logs with pagination"
// @Failure 400 {object} ErrorResponse "Invalid project ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{projectId}/notes [get]
func (h *NoteHandler) GetProjectNotes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Project ID is required"})
		return
	}

	// Parse pagination parameters
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	notes, total, err := h.noteService.GetProjectNotes(c.Request.Context(), userID, projectID, offset, limit)
	if err != nil {
		h.logger.Errorf("get project notes error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListNotesResponse{
		Notes: notes,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	})
}

// CreateProjectNote handles creating a note for a specific project (project log entry)
// @Summary Create project note/log
// @Description Create a new note/log entry for a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param request body note.CreateNoteRequest true "Note creation data"
// @Success 201 {object} CreateNoteResponse "Project note created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{projectId}/notes [post]
func (h *NoteHandler) CreateProjectNote(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Project ID is required"})
		return
	}

	var req note.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	noteResponse, err := h.noteService.CreateProjectNote(c.Request.Context(), userID, projectID, req)
	if err != nil {
		switch err {
		case note.ErrInvalidNoteData:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid note data"})
		default:
			h.logger.Errorf("create project note error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, CreateNoteResponse{
		Message: "Project note created successfully",
		Note:    *noteResponse,
	})
}

// RegisterNoteRoutes registers all note-related routes
func (h *NoteHandler) RegisterNoteRoutes(r *gin.RouterGroup, userService user.UserService) {
	// Protected routes (authentication required)
	protected := r.Group("/notes")
	protected.Use(AuthMiddleware(userService, h.logger))
	{
		protected.POST("", h.CreateNote)
		protected.GET("", h.ListUserNotes)
		protected.GET("/search", h.SearchNotes)
		protected.GET("/tags", h.GetNotesByTags)
		protected.GET("/:id", h.GetNote)
		protected.PUT("/:id", h.UpdateNote)
		protected.DELETE("/:id", h.DeleteNote)
	}

	// Admin routes (admin role required)
	admin := r.Group("/admin/notes")
	admin.Use(AuthMiddleware(userService, h.logger), AdminMiddleware())
	{
		admin.GET("", h.ListAllNotes)
	}
}
