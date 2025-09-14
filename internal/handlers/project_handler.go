package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/project"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	projectService project.ProjectService
	logger         *Logger.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService project.ProjectService, logger *Logger.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		logger:         logger,
	}
}

// CreateProject handles project creation
// @Summary Create a new project
// @Description Create a new project for the authenticated user
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body project.CreateProjectRequest true "Project creation data"
// @Success 201 {object} CreateProjectResponse "Project created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req project.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	projectResponse, err := h.projectService.CreateProject(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Errorf("create project error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, CreateProjectResponse{
		Message: "Project created successfully",
		Project: *projectResponse,
	})
}

// GetProject handles getting a specific project
// @Summary Get project by ID
// @Description Get a specific project by ID (user can only access their own projects)
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} ProjectResponse "Project data"
// @Failure 400 {object} ErrorResponse "Invalid project ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
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

	projectResponse, err := h.projectService.GetProject(c.Request.Context(), userID, projectID)
	if err != nil {
		switch err {
		case project.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Project not found"})
		case project.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("get project error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{
		Project: *projectResponse,
	})
}

// UpdateProject handles updating a project
// @Summary Update project
// @Description Update a specific project (user can only update their own projects)
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body project.UpdateProjectRequest true "Project update data"
// @Success 200 {object} UpdateProjectResponse "Project updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
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

	var req project.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	projectResponse, err := h.projectService.UpdateProject(c.Request.Context(), userID, projectID, req)
	if err != nil {
		switch err {
		case project.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Project not found"})
		case project.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("update project error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateProjectResponse{
		Message: "Project updated successfully",
		Project: *projectResponse,
	})
}

// DeleteProject handles project deletion
// @Summary Delete project
// @Description Delete a specific project (user can only delete their own projects)
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} SuccessResponse "Project deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid project ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
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

	err := h.projectService.DeleteProject(c.Request.Context(), userID, projectID)
	if err != nil {
		switch err {
		case project.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Project not found"})
		case project.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("delete project error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Project deleted successfully",
	})
}

// ListUserProjects handles listing user's projects with filtering
// @Summary List user projects
// @Description List all projects for the authenticated user with optional filtering
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(planned,in_progress,blocked,done,archived)
// @Param priority query string false "Filter by priority" Enums(low,med,high,urgent)
// @Param tags query []string false "Filter by tags (comma-separated)"
// @Param offset query int false "Number of projects to skip" default(0)
// @Param limit query int false "Number of projects to return" default(20)
// @Success 200 {object} ListProjectsResponse "List of projects with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects [get]
func (h *ProjectHandler) ListUserProjects(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	// Parse query parameters
	var filters project.ListProjectsRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set default limits
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	projects, total, err := h.projectService.ListUserProjects(c.Request.Context(), userID, filters)
	if err != nil {
		h.logger.Errorf("list user projects error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListProjectsResponse{
		Projects: projects,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// AddProgressEvent handles adding a progress event to a project
// @Summary Add progress event
// @Description Add a progress event to a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body project.AddProgressEventRequest true "Progress event data"
// @Success 200 {object} UpdateProjectResponse "Progress event added successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id}/progress [post]
func (h *ProjectHandler) AddProgressEvent(c *gin.Context) {
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

	var req project.AddProgressEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	projectResponse, err := h.projectService.AddProgressEvent(c.Request.Context(), userID, projectID, req)
	if err != nil {
		switch err {
		case project.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Project not found"})
		case project.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("add progress event error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateProjectResponse{
		Message: "Progress event added successfully",
		Project: *projectResponse,
	})
}

// UpdateProjectStatus handles updating project status
// @Summary Update project status
// @Description Update the status of a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body UpdateProjectStatusRequest true "Status update data"
// @Success 200 {object} UpdateProjectResponse "Project status updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{id}/status [put]
func (h *ProjectHandler) UpdateProjectStatus(c *gin.Context) {
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

	var req UpdateProjectStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	projectResponse, err := h.projectService.UpdateProjectStatus(c.Request.Context(), userID, projectID, req.Status)
	if err != nil {
		switch err {
		case project.ErrProjectNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Project not found"})
		case project.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("update project status error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateProjectResponse{
		Message: "Project status updated successfully",
		Project: *projectResponse,
	})
}

// ListAllProjects handles listing all projects (admin endpoint)
// @Summary List all projects (Admin)
// @Description List all projects in the system with optional filtering (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(planned,in_progress,blocked,done,archived)
// @Param priority query string false "Filter by priority" Enums(low,med,high,urgent)
// @Param tags query []string false "Filter by tags (comma-separated)"
// @Param offset query int false "Number of projects to skip" default(0)
// @Param limit query int false "Number of projects to return" default(20)
// @Success 200 {object} ListProjectsResponse "List of projects with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/projects [get]
func (h *ProjectHandler) ListAllProjects(c *gin.Context) {
	// Parse query parameters
	var filters project.ListProjectsRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set default limits
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	projects, total, err := h.projectService.ListAllProjects(c.Request.Context(), filters)
	if err != nil {
		h.logger.Errorf("list all projects error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListProjectsResponse{
		Projects: projects,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// RegisterProjectRoutes registers all project-related routes
func (h *ProjectHandler) RegisterProjectRoutes(r *gin.RouterGroup, userService user.UserService, noteHandler *NoteHandler) {
	// Protected routes (authentication required)
	protected := r.Group("/projects")
	protected.Use(AuthMiddleware(userService, h.logger))
	{
		protected.POST("", h.CreateProject)
		protected.GET("", h.ListUserProjects)
		protected.GET("/:id", h.GetProject)
		protected.PUT("/:id", h.UpdateProject)
		protected.DELETE("/:id", h.DeleteProject)
		protected.POST("/:id/progress", h.AddProgressEvent)
		protected.PUT("/:id/status", h.UpdateProjectStatus)

		// Project notes/logs endpoints (if noteHandler is provided)
		if noteHandler != nil {
			protected.GET("/:id/notes", noteHandler.GetProjectNotes)
			protected.POST("/:id/notes", noteHandler.CreateProjectNote)
		}
	}

	// Admin routes (admin role required)
	admin := r.Group("/admin/projects")
	admin.Use(AuthMiddleware(userService, h.logger), AdminMiddleware())
	{
		admin.GET("", h.ListAllProjects)
	}
}
