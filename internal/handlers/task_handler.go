package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	taskService task.TaskService
	logger      *Logger.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(taskService task.TaskService, logger *Logger.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// CreateTask handles task creation
// @Summary Create a new task
// @Description Create a new task for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body task.CreateTaskRequest true "Task creation data"
// @Success 201 {object} CreateTaskResponse "Task created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req task.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	taskResponse, err := h.taskService.CreateTask(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case task.ErrInvalidTaskData:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid task data"})
		default:
			h.logger.Errorf("create task error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, CreateTaskResponse{
		Message: "Task created successfully",
		Task:    *taskResponse,
	})
}

// GetTask handles getting a specific task
// @Summary Get task by ID
// @Description Get a specific task by ID (user can only access their own tasks)
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} TaskResponse "Task data"
// @Failure 400 {object} ErrorResponse "Invalid task ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	taskResponse, err := h.taskService.GetTask(c.Request.Context(), userID, taskID)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("get task error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, TaskResponse{
		Task: *taskResponse,
	})
}

// UpdateTask handles updating a task
// @Summary Update task
// @Description Update a specific task (user can only update their own tasks)
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Param request body task.UpdateTaskRequest true "Task update data"
// @Success 200 {object} UpdateTaskResponse "Task updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	var req task.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	taskResponse, err := h.taskService.UpdateTask(c.Request.Context(), userID, taskID, req)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		case task.ErrInvalidTaskData:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid task data"})
		default:
			h.logger.Errorf("update task error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateTaskResponse{
		Message: "Task updated successfully",
		Task:    *taskResponse,
	})
}

// DeleteTask handles task deletion
// @Summary Delete task
// @Description Delete a specific task (user can only delete their own tasks)
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} SuccessResponse "Task deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid task ID"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	err := h.taskService.DeleteTask(c.Request.Context(), userID, taskID)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("delete task error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Task deleted successfully",
	})
}

// UpdateTaskStatus handles updating task status
// @Summary Update task status
// @Description Update the status of a specific task
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Param request body task.UpdateTaskStatusRequest true "Status update data"
// @Success 200 {object} UpdateTaskResponse "Task status updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id}/status [put]
func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	var req task.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	taskResponse, err := h.taskService.UpdateTaskStatus(c.Request.Context(), userID, taskID, req)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		case task.ErrInvalidTaskStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid task status"})
		case task.ErrTaskAlreadyCompleted:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task is already completed"})
		case task.ErrTaskAlreadyCancelled:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task is already cancelled"})
		default:
			h.logger.Errorf("update task status error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateTaskResponse{
		Message: "Task status updated successfully",
		Task:    *taskResponse,
	})
}

// MarkTaskCompleted handles marking a task as completed
// @Summary Mark task as completed
// @Description Mark a specific task as completed
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} UpdateTaskResponse "Task marked as completed"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id}/complete [post]
func (h *TaskHandler) MarkTaskCompleted(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	taskResponse, err := h.taskService.MarkTaskCompleted(c.Request.Context(), userID, taskID)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		case task.ErrTaskAlreadyCompleted:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task is already completed"})
		default:
			h.logger.Errorf("mark task completed error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateTaskResponse{
		Message: "Task marked as completed",
		Task:    *taskResponse,
	})
}

// MarkTaskCancelled handles marking a task as cancelled
// @Summary Mark task as cancelled
// @Description Mark a specific task as cancelled
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} UpdateTaskResponse "Task marked as cancelled"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{id}/cancel [post]
func (h *TaskHandler) MarkTaskCancelled(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task ID is required"})
		return
	}

	taskResponse, err := h.taskService.MarkTaskCancelled(c.Request.Context(), userID, taskID)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		case task.ErrTaskAlreadyCancelled:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Task is already cancelled"})
		default:
			h.logger.Errorf("mark task cancelled error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateTaskResponse{
		Message: "Task marked as cancelled",
		Task:    *taskResponse,
	})
}

// ListUserTasks handles listing user's tasks with filtering
// @Summary List user tasks
// @Description List all tasks for the authenticated user with optional filtering
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (pending, done, cancelled)"
// @Param priority query int false "Filter by priority (1-5)"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param isRecurring query bool false "Filter by recurring status"
// @Param isOverdue query bool false "Filter overdue tasks"
// @Param fromDate query string false "Filter from date (RFC3339)"
// @Param toDate query string false "Filter to date (RFC3339)"
// @Param search query string false "Search in task title and description"
// @Param orderBy query string false "Order by field (scheduledAt, dueAt, priority, createdAt)" default(createdAt)
// @Param order query string false "Order direction (asc, desc)" default(desc)
// @Param offset query int false "Number of tasks to skip" default(0)
// @Param limit query int false "Number of tasks to return" default(20)
// @Success 200 {object} ListTasksResponse "List of tasks with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks [get]
func (h *TaskHandler) ListUserTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	// Parse query parameters
	var filters task.ListTasksRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Parse tags from comma-separated string
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		filters.Tags = strings.Split(tagsStr, ",")
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

	tasks, total, err := h.taskService.ListUserTasks(c.Request.Context(), userID, filters)
	if err != nil {
		h.logger.Errorf("list user tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListTasksResponse{
		Tasks: tasks,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// GetTasksDueToday handles getting tasks due today
// @Summary Get tasks due today
// @Description Get all tasks due today for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []task.TaskResponse "List of tasks due today"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/due-today [get]
func (h *TaskHandler) GetTasksDueToday(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	tasks, err := h.taskService.GetTasksDueToday(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("get tasks due today error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetOverdueTasks handles getting overdue tasks
// @Summary Get overdue tasks
// @Description Get all overdue tasks for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []task.TaskResponse "List of overdue tasks"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/overdue [get]
func (h *TaskHandler) GetOverdueTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	tasks, err := h.taskService.GetOverdueTasks(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("get overdue tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetUpcomingTasks handles getting upcoming tasks
// @Summary Get upcoming tasks
// @Description Get upcoming tasks for the authenticated user within specified days
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param days query int false "Number of days to look ahead" default(7)
// @Success 200 {object} []task.TaskResponse "List of upcoming tasks"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/upcoming [get]
func (h *TaskHandler) GetUpcomingTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 7
	}

	tasks, err := h.taskService.GetUpcomingTasks(c.Request.Context(), userID, days)
	if err != nil {
		h.logger.Errorf("get upcoming tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetCalendarTasks handles getting tasks for calendar view
// @Summary Get calendar tasks
// @Description Get tasks for calendar view within specified date range
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fromDate query string true "Start date (RFC3339)"
// @Param toDate query string true "End date (RFC3339)"
// @Success 200 {object} []task.CalendarTaskResponse "List of calendar tasks"
// @Failure 400 {object} ErrorResponse "Invalid date parameters"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/calendar [get]
func (h *TaskHandler) GetCalendarTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	if fromDateStr == "" || toDateStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "fromDate and toDate parameters are required"})
		return
	}

	fromDate, err := time.Parse(time.RFC3339, fromDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid fromDate format, use RFC3339"})
		return
	}

	toDate, err := time.Parse(time.RFC3339, toDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid toDate format, use RFC3339"})
		return
	}

	tasks, err := h.taskService.GetCalendarTasks(c.Request.Context(), userID, fromDate, toDate)
	if err != nil {
		h.logger.Errorf("get calendar tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// SearchTasks handles searching tasks
// @Summary Search tasks
// @Description Search tasks by content and/or other criteria for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param status query string false "Filter by status"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param offset query int false "Number of tasks to skip" default(0)
// @Param limit query int false "Number of tasks to return" default(20)
// @Success 200 {object} SearchTasksResponse "Search results with pagination"
// @Failure 400 {object} ErrorResponse "Search query is required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/search [get]
func (h *TaskHandler) SearchTasks(c *gin.Context) {
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

	// Parse filters
	var filters task.ListTasksRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Parse tags
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		filters.Tags = strings.Split(tagsStr, ",")
		for i, tag := range filters.Tags {
			filters.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Set defaults
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	tasks, total, err := h.taskService.SearchTasks(c.Request.Context(), userID, query, filters)
	if err != nil {
		h.logger.Errorf("search tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, SearchTasksResponse{
		Tasks: tasks,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
		Query: query,
	})
}

// GetTasksByTags handles getting tasks by tags
// @Summary Get tasks by tags
// @Description Get tasks filtered by specific tags for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tags query string true "Tags to filter by (comma-separated)"
// @Param offset query int false "Number of tasks to skip" default(0)
// @Param limit query int false "Number of tasks to return" default(20)
// @Success 200 {object} ListTasksResponse "List of tasks with pagination"
// @Failure 400 {object} ErrorResponse "Tags parameter is required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/tags [get]
func (h *TaskHandler) GetTasksByTags(c *gin.Context) {
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

	tasks, total, err := h.taskService.GetTasksByTags(c.Request.Context(), userID, tags, offset, limit)
	if err != nil {
		h.logger.Errorf("get tasks by tags error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListTasksResponse{
		Tasks: tasks,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	})
}

// GetRecurringTasks handles getting recurring tasks
// @Summary Get recurring tasks
// @Description Get all recurring tasks for the authenticated user
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []task.TaskResponse "List of recurring tasks"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/recurring [get]
func (h *TaskHandler) GetRecurringTasks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	tasks, err := h.taskService.GetRecurringTasks(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("get recurring tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetTaskInstances handles getting instances of a recurring task
// @Summary Get task instances
// @Description Get all instances of a recurring task
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param parentId path string true "Parent Task ID"
// @Success 200 {object} []task.TaskResponse "List of task instances"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Unauthorized access"
// @Failure 404 {object} ErrorResponse "Parent task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/{parentId}/instances [get]
func (h *TaskHandler) GetTaskInstances(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	parentID := c.Param("parentId")
	if parentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Parent task ID is required"})
		return
	}

	tasks, err := h.taskService.GetTaskInstances(c.Request.Context(), userID, parentID)
	if err != nil {
		switch err {
		case task.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Parent task not found"})
		case task.ErrUnauthorizedAccess:
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Unauthorized access"})
		default:
			h.logger.Errorf("get task instances error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// BulkUpdateStatus handles bulk status updates
// @Summary Bulk update task status
// @Description Update status for multiple tasks at once
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BulkUpdateStatusRequest true "Bulk update request"
// @Success 200 {object} SuccessResponse "Tasks updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /tasks/bulk/status [put]
func (h *TaskHandler) BulkUpdateStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req BulkUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	if len(req.TaskIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No task IDs provided"})
		return
	}

	err := h.taskService.BulkUpdateStatus(c.Request.Context(), userID, req.TaskIDs, req.Status)
	if err != nil {
		switch err {
		case task.ErrInvalidTaskStatus:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid task status"})
		default:
			h.logger.Errorf("bulk update status error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Tasks updated successfully",
	})
}

// ListAllTasks handles listing all tasks (admin endpoint)
// @Summary List all tasks (Admin)
// @Description List all tasks in the system with optional filtering (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param priority query int false "Filter by priority"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param search query string false "Search in task content"
// @Param orderBy query string false "Order by field" default(createdAt)
// @Param order query string false "Order direction (asc, desc)" default(desc)
// @Param offset query int false "Number of tasks to skip" default(0)
// @Param limit query int false "Number of tasks to return" default(20)
// @Success 200 {object} ListTasksResponse "List of tasks with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/tasks [get]
func (h *TaskHandler) ListAllTasks(c *gin.Context) {
	// Parse query parameters
	var filters task.ListTasksRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Parse tags from comma-separated string
	tagsStr := c.Query("tags")
	if tagsStr != "" {
		filters.Tags = strings.Split(tagsStr, ",")
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

	tasks, total, err := h.taskService.ListAllTasks(c.Request.Context(), filters)
	if err != nil {
		h.logger.Errorf("list all tasks error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListTasksResponse{
		Tasks: tasks,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: filters.Offset,
			Limit:  filters.Limit,
		},
	})
}

// RegisterTaskRoutes registers all task-related routes
func (h *TaskHandler) RegisterTaskRoutes(r *gin.RouterGroup, userService user.UserService) {
	// Protected routes (authentication required)
	protected := r.Group("/tasks")
	protected.Use(AuthMiddleware(userService, h.logger))
	{
		protected.POST("", h.CreateTask)
		protected.GET("", h.ListUserTasks)
		protected.GET("/search", h.SearchTasks)
		protected.GET("/tags", h.GetTasksByTags)
		protected.GET("/due-today", h.GetTasksDueToday)
		protected.GET("/overdue", h.GetOverdueTasks)
		protected.GET("/upcoming", h.GetUpcomingTasks)
		protected.GET("/calendar", h.GetCalendarTasks)
		protected.GET("/recurring", h.GetRecurringTasks)
		protected.PUT("/bulk/status", h.BulkUpdateStatus)
		protected.GET("/instances/:parentId", h.GetTaskInstances) // Move instances route before :id route
		protected.GET("/:id", h.GetTask)
		protected.PUT("/:id", h.UpdateTask)
		protected.DELETE("/:id", h.DeleteTask)
		protected.PUT("/:id/status", h.UpdateTaskStatus)
		protected.POST("/:id/complete", h.MarkTaskCompleted)
		protected.POST("/:id/cancel", h.MarkTaskCancelled)
	}

	// Admin routes (admin role required)
	admin := r.Group("/admin/tasks")
	admin.Use(AuthMiddleware(userService, h.logger), AdminMiddleware())
	{
		admin.GET("", h.ListAllTasks)
	}
}
