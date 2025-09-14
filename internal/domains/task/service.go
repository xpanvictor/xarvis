package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// Common errors
var (
	ErrTaskNotFound         = errors.New("task not found")
	ErrUnauthorizedAccess   = errors.New("unauthorized access to task")
	ErrInvalidTaskData      = errors.New("invalid task data")
	ErrInvalidTaskStatus    = errors.New("invalid task status")
	ErrInvalidRecurrence    = errors.New("invalid recurrence configuration")
	ErrTaskAlreadyCompleted = errors.New("task is already completed")
	ErrTaskAlreadyCancelled = errors.New("task is already cancelled")
)

// TaskService defines the interface for task business logic
type TaskService interface {
	// Basic task management
	CreateTask(ctx context.Context, userID string, req CreateTaskRequest) (*TaskResponse, error)
	GetTask(ctx context.Context, userID, taskID string) (*TaskResponse, error)
	UpdateTask(ctx context.Context, userID, taskID string, req UpdateTaskRequest) (*TaskResponse, error)
	DeleteTask(ctx context.Context, userID, taskID string) error

	// Task status management
	UpdateTaskStatus(ctx context.Context, userID, taskID string, req UpdateTaskStatusRequest) (*TaskResponse, error)
	MarkTaskCompleted(ctx context.Context, userID, taskID string) (*TaskResponse, error)
	MarkTaskCancelled(ctx context.Context, userID, taskID string) (*TaskResponse, error)

	// Task listing and filtering
	ListUserTasks(ctx context.Context, userID string, filters ListTasksRequest) ([]TaskResponse, int64, error)
	GetTasksByStatus(ctx context.Context, userID string, status TaskStatus, offset, limit int) ([]TaskResponse, int64, error)
	GetTasksByDateRange(ctx context.Context, userID string, fromDate, toDate time.Time, offset, limit int) ([]TaskResponse, int64, error)

	// Scheduling and calendar operations
	GetTasksDueToday(ctx context.Context, userID string) ([]TaskResponse, error)
	GetOverdueTasks(ctx context.Context, userID string) ([]TaskResponse, error)
	GetUpcomingTasks(ctx context.Context, userID string, days int) ([]TaskResponse, error)
	GetCalendarTasks(ctx context.Context, userID string, fromDate, toDate time.Time) ([]CalendarTaskResponse, error)

	// Search functionality
	SearchTasks(ctx context.Context, userID, query string, filters ListTasksRequest) ([]TaskResponse, int64, error)
	GetTasksByTags(ctx context.Context, userID string, tags []string, offset, limit int) ([]TaskResponse, int64, error)

	// Recurring task management
	GetRecurringTasks(ctx context.Context, userID string) ([]TaskResponse, error)
	GetTaskInstances(ctx context.Context, userID, parentTaskID string) ([]TaskResponse, error)
	CreateRecurringInstance(ctx context.Context, parentTaskID string, scheduledAt time.Time) (*TaskResponse, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, userID string, taskIDs []string, status TaskStatus) error

	// Admin operations
	ListAllTasks(ctx context.Context, filters ListTasksRequest) ([]TaskResponse, int64, error)
	GetUserTasks(ctx context.Context, userID string) ([]TaskResponse, error)

	// Execution operations (for scheduler)
	GetTasksToExecute(ctx context.Context, beforeTime time.Time, limit int) ([]TaskResponse, error)
	ExecuteTask(ctx context.Context, taskID string) error
}

type taskService struct {
	repository TaskRepository
	logger     *Logger.Logger
}

// CreateTask implements TaskService
func (s *taskService) CreateTask(ctx context.Context, userID string, req CreateTaskRequest) (*TaskResponse, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create task using domain constructor
	task := NewTask(userUUID, req)

	if err := s.repository.Create(task); err != nil {
		s.logger.Errorf("error creating task: %v", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.Infof("task created successfully: %s for user %s", task.ID, userID)
	response := task.ToResponse()
	return &response, nil
}

// GetTask implements TaskService
func (s *taskService) GetTask(ctx context.Context, userID, taskID string) (*TaskResponse, error) {
	task, err := s.repository.GetByID(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		s.logger.Errorf("error getting task: %v", err)
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Check if user has access to this task
	if task.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	response := task.ToResponse()
	return &response, nil
}

// UpdateTask implements TaskService
func (s *taskService) UpdateTask(ctx context.Context, userID, taskID string, req UpdateTaskRequest) (*TaskResponse, error) {
	// First check if task exists and user has access
	existing, err := s.repository.GetByID(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if existing.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Validate update data
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Update the task
	updatedTask, err := s.repository.Update(taskID, req)
	if err != nil {
		s.logger.Errorf("error updating task: %v", err)
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Infof("task updated successfully: %s", taskID)
	response := updatedTask.ToResponse()
	return &response, nil
}

// DeleteTask implements TaskService
func (s *taskService) DeleteTask(ctx context.Context, userID, taskID string) error {
	// First check if task exists and user has access
	existing, err := s.repository.GetByID(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	if existing.UserID.String() != userID {
		return ErrUnauthorizedAccess
	}

	if err := s.repository.Delete(taskID); err != nil {
		s.logger.Errorf("error deleting task: %v", err)
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.logger.Infof("task deleted successfully: %s", taskID)
	return nil
}

// UpdateTaskStatus implements TaskService
func (s *taskService) UpdateTaskStatus(ctx context.Context, userID, taskID string, req UpdateTaskStatusRequest) (*TaskResponse, error) {
	// Validate status
	if !req.Status.IsValid() {
		return nil, ErrInvalidTaskStatus
	}

	// First check if task exists and user has access
	existing, err := s.repository.GetByID(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if existing.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Check for invalid status transitions
	if existing.Status == StatusDone && req.Status == StatusDone {
		return nil, ErrTaskAlreadyCompleted
	}
	if existing.Status == StatusCancelled && req.Status == StatusCancelled {
		return nil, ErrTaskAlreadyCancelled
	}

	// Update the task status
	updatedTask, err := s.repository.UpdateStatus(taskID, req.Status)
	if err != nil {
		s.logger.Errorf("error updating task status: %v", err)
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	s.logger.Infof("task status updated successfully: %s to %s", taskID, req.Status)
	response := updatedTask.ToResponse()
	return &response, nil
}

// MarkTaskCompleted implements TaskService
func (s *taskService) MarkTaskCompleted(ctx context.Context, userID, taskID string) (*TaskResponse, error) {
	req := UpdateTaskStatusRequest{Status: StatusDone}
	return s.UpdateTaskStatus(ctx, userID, taskID, req)
}

// MarkTaskCancelled implements TaskService
func (s *taskService) MarkTaskCancelled(ctx context.Context, userID, taskID string) (*TaskResponse, error) {
	req := UpdateTaskStatusRequest{Status: StatusCancelled}
	return s.UpdateTaskStatus(ctx, userID, taskID, req)
}

// ListUserTasks implements TaskService
func (s *taskService) ListUserTasks(ctx context.Context, userID string, filters ListTasksRequest) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Errorf("error listing user tasks: %v", err)
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetTasksByStatus implements TaskService
func (s *taskService) GetTasksByStatus(ctx context.Context, userID string, status TaskStatus, offset, limit int) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.GetUserTasksByStatus(userID, status, offset, limit)
	if err != nil {
		s.logger.Errorf("error getting tasks by status: %v", err)
		return nil, 0, fmt.Errorf("failed to get tasks by status: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetTasksByDateRange implements TaskService
func (s *taskService) GetTasksByDateRange(ctx context.Context, userID string, fromDate, toDate time.Time, offset, limit int) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.GetUserTasksByDateRange(userID, fromDate, toDate, offset, limit)
	if err != nil {
		s.logger.Errorf("error getting tasks by date range: %v", err)
		return nil, 0, fmt.Errorf("failed to get tasks by date range: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetTasksDueToday implements TaskService
func (s *taskService) GetTasksDueToday(ctx context.Context, userID string) ([]TaskResponse, error) {
	tasks, err := s.repository.GetTasksDueToday(userID)
	if err != nil {
		s.logger.Errorf("error getting tasks due today: %v", err)
		return nil, fmt.Errorf("failed to get tasks due today: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// GetOverdueTasks implements TaskService
func (s *taskService) GetOverdueTasks(ctx context.Context, userID string) ([]TaskResponse, error) {
	tasks, err := s.repository.GetOverdueTasks(userID)
	if err != nil {
		s.logger.Errorf("error getting overdue tasks: %v", err)
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// GetUpcomingTasks implements TaskService
func (s *taskService) GetUpcomingTasks(ctx context.Context, userID string, days int) ([]TaskResponse, error) {
	tasks, err := s.repository.GetUpcomingTasks(userID, days)
	if err != nil {
		s.logger.Errorf("error getting upcoming tasks: %v", err)
		return nil, fmt.Errorf("failed to get upcoming tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// GetCalendarTasks implements TaskService
func (s *taskService) GetCalendarTasks(ctx context.Context, userID string, fromDate, toDate time.Time) ([]CalendarTaskResponse, error) {
	tasks, err := s.repository.GetCalendarTasks(userID, fromDate, toDate)
	if err != nil {
		s.logger.Errorf("error getting calendar tasks: %v", err)
		return nil, fmt.Errorf("failed to get calendar tasks: %w", err)
	}

	return tasks, nil
}

// SearchTasks implements TaskService
func (s *taskService) SearchTasks(ctx context.Context, userID, query string, filters ListTasksRequest) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.Search(userID, query, filters)
	if err != nil {
		s.logger.Errorf("error searching tasks: %v", err)
		return nil, 0, fmt.Errorf("failed to search tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetTasksByTags implements TaskService
func (s *taskService) GetTasksByTags(ctx context.Context, userID string, tags []string, offset, limit int) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.GetByTags(userID, tags, offset, limit)
	if err != nil {
		s.logger.Errorf("error getting tasks by tags: %v", err)
		return nil, 0, fmt.Errorf("failed to get tasks by tags: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetRecurringTasks implements TaskService
func (s *taskService) GetRecurringTasks(ctx context.Context, userID string) ([]TaskResponse, error) {
	tasks, err := s.repository.GetRecurringTasks()
	if err != nil {
		s.logger.Errorf("error getting recurring tasks: %v", err)
		return nil, fmt.Errorf("failed to get recurring tasks: %w", err)
	}

	// Filter by user ID
	var userTasks []Task
	for _, task := range tasks {
		if task.UserID.String() == userID {
			userTasks = append(userTasks, task)
		}
	}

	responses := make([]TaskResponse, len(userTasks))
	for i, task := range userTasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// GetTaskInstances implements TaskService
func (s *taskService) GetTaskInstances(ctx context.Context, userID, parentTaskID string) ([]TaskResponse, error) {
	// First check if parent task exists and user has access
	parentTask, err := s.repository.GetByID(parentTaskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	if parentTask.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	tasks, err := s.repository.GetChildTasks(parentTaskID)
	if err != nil {
		s.logger.Errorf("error getting task instances: %v", err)
		return nil, fmt.Errorf("failed to get task instances: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// CreateRecurringInstance implements TaskService
func (s *taskService) CreateRecurringInstance(ctx context.Context, parentTaskID string, scheduledAt time.Time) (*TaskResponse, error) {
	// Get parent task
	parentTask, err := s.repository.GetByID(parentTaskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	if !parentTask.IsRecurring {
		return nil, fmt.Errorf("task is not recurring")
	}

	newTask, err := s.repository.CreateRecurringInstance(parentTask, scheduledAt)
	if err != nil {
		s.logger.Errorf("error creating recurring instance: %v", err)
		return nil, fmt.Errorf("failed to create recurring instance: %w", err)
	}

	s.logger.Infof("recurring task instance created: %s from parent %s", newTask.ID, parentTaskID)
	response := newTask.ToResponse()
	return &response, nil
}

// BulkUpdateStatus implements TaskService
func (s *taskService) BulkUpdateStatus(ctx context.Context, userID string, taskIDs []string, status TaskStatus) error {
	// Validate status
	if !status.IsValid() {
		return ErrInvalidTaskStatus
	}

	// Verify user has access to all tasks
	for _, taskID := range taskIDs {
		task, err := s.repository.GetByID(taskID)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return fmt.Errorf("task %s not found", taskID)
			}
			return fmt.Errorf("failed to verify task %s: %w", taskID, err)
		}

		if task.UserID.String() != userID {
			return fmt.Errorf("unauthorized access to task %s", taskID)
		}
	}

	if err := s.repository.BulkUpdateStatus(taskIDs, status); err != nil {
		s.logger.Errorf("error bulk updating task status: %v", err)
		return fmt.Errorf("failed to bulk update task status: %w", err)
	}

	s.logger.Infof("bulk updated %d tasks to status %s for user %s", len(taskIDs), status, userID)
	return nil
}

// ListAllTasks implements TaskService (admin only)
func (s *taskService) ListAllTasks(ctx context.Context, filters ListTasksRequest) ([]TaskResponse, int64, error) {
	tasks, total, err := s.repository.List(filters)
	if err != nil {
		s.logger.Errorf("error listing all tasks: %v", err)
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, total, nil
}

// GetUserTasks implements TaskService (admin only)
func (s *taskService) GetUserTasks(ctx context.Context, userID string) ([]TaskResponse, error) {
	tasks, err := s.repository.GetTasksByUser(userID)
	if err != nil {
		s.logger.Errorf("error getting user tasks: %v", err)
		return nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// GetTasksToExecute implements TaskService (for scheduler)
func (s *taskService) GetTasksToExecute(ctx context.Context, beforeTime time.Time, limit int) ([]TaskResponse, error) {
	tasks, err := s.repository.GetTasksToExecute(beforeTime, limit)
	if err != nil {
		s.logger.Errorf("error getting tasks to execute: %v", err)
		return nil, fmt.Errorf("failed to get tasks to execute: %w", err)
	}

	responses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = task.ToResponse()
	}

	return responses, nil
}

// ExecuteTask implements TaskService (for scheduler/brain system)
func (s *taskService) ExecuteTask(ctx context.Context, taskID string) error {
	// Get the task
	task, err := s.repository.GetByID(taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to get task for execution: %w", err)
	}

	// Check if task can be executed
	if !task.CanExecute() {
		return fmt.Errorf("task cannot be executed in current state")
	}

	// TODO: Send task to brain system for execution
	// This is where we would integrate with the brain system
	s.logger.Infof("executing task: %s - %s", task.ID, task.Title)

	// For now, we'll just mark the task as completed
	// In the future, the brain system will handle execution and status updates
	_, err = s.repository.UpdateStatus(taskID, StatusDone)
	if err != nil {
		s.logger.Errorf("error updating task status after execution: %v", err)
		return fmt.Errorf("failed to update task status: %w", err)
	}

	s.logger.Infof("task executed successfully: %s", taskID)
	return nil
}

// Validation methods
func (s *taskService) validateCreateRequest(req CreateTaskRequest) error {
	if req.Title == "" {
		return ErrInvalidTaskData
	}

	if req.Priority < 1 || req.Priority > 5 {
		return ErrInvalidTaskData
	}

	if req.DueAt != nil && req.ScheduledAt != nil && req.DueAt.Before(*req.ScheduledAt) {
		return fmt.Errorf("due date cannot be before scheduled date")
	}

	if req.IsRecurring {
		if err := s.validateRecurrenceConfig(req.RecurrenceConfig); err != nil {
			return err
		}
	}

	return nil
}

func (s *taskService) validateUpdateRequest(req UpdateTaskRequest) error {
	if req.Title != nil && *req.Title == "" {
		return ErrInvalidTaskData
	}

	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 5) {
		return ErrInvalidTaskData
	}

	if req.DueAt != nil && req.ScheduledAt != nil && req.DueAt.Before(*req.ScheduledAt) {
		return fmt.Errorf("due date cannot be before scheduled date")
	}

	if req.IsRecurring != nil && *req.IsRecurring {
		if err := s.validateRecurrenceConfig(req.RecurrenceConfig); err != nil {
			return err
		}
	}

	return nil
}

func (s *taskService) validateRecurrenceConfig(config *RecurrenceConfig) error {
	if config == nil {
		return ErrInvalidRecurrence
	}

	if !config.Type.IsValid() {
		return ErrInvalidRecurrence
	}

	if config.Type == RecurrenceCustom && config.Interval <= 0 {
		return fmt.Errorf("custom recurrence requires positive interval")
	}

	if config.MaxOccurrences != nil && *config.MaxOccurrences <= 0 {
		return fmt.Errorf("max occurrences must be positive")
	}

	return nil
}

// NewTaskService creates a new task service
func NewTaskService(repository TaskRepository, logger *Logger.Logger) TaskService {
	return &taskService{
		repository: repository,
		logger:     logger,
	}
}
