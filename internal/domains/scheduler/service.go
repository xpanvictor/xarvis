package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/runtime/brain"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
)

// AsynqSchedulerService implements SchedulerService using asynq
type AsynqSchedulerService struct {
	client           *asynq.Client
	server           *asynq.Server
	mux              *asynq.ServeMux
	logger           *Logger.Logger
	taskService      task.TaskService
	brainIntegration BrainSystemIntegration

	// Dependencies for brain system integration
	deviceRegistry     registry.DeviceRegistry
	llmRouter          *router.Mux
	brainSystemFactory *brain.BrainSystemFactory
}

// AsynqSchedulerConfig holds configuration for the scheduler
type AsynqSchedulerConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int
	Queues        map[string]int
}

// NewAsynqSchedulerService creates a new scheduler service
func NewAsynqSchedulerService(
	config AsynqSchedulerConfig,
	logger *Logger.Logger,
	taskService task.TaskService,
	deviceRegistry registry.DeviceRegistry,
	llmRouter *router.Mux,
	brainSystemFactory *brain.BrainSystemFactory,
) *AsynqSchedulerService {

	redisOpt := asynq.RedisClientOpt{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	}

	client := asynq.NewClient(redisOpt)

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: config.Concurrency,
		Queues:      config.Queues,
		Logger:      NewAsynqLogger(logger),
	})

	mux := asynq.NewServeMux()

	service := &AsynqSchedulerService{
		client:             client,
		server:             server,
		mux:                mux,
		logger:             logger,
		taskService:        taskService,
		deviceRegistry:     deviceRegistry,
		llmRouter:          llmRouter,
		brainSystemFactory: brainSystemFactory,
	}

	// Initialize brain integration with a temp brain system for initial setup
	// In practice, brain systems will be created per task execution
	tempBrainSystem := brainSystemFactory.CreateBrainSystem()
	service.brainIntegration = NewBrainIntegration(tempBrainSystem, deviceRegistry, logger)

	// Register job handlers
	service.registerHandlers()

	return service
}

// registerHandlers registers asynq job handlers
func (s *AsynqSchedulerService) registerHandlers() {
	s.mux.HandleFunc(string(JobTypeTaskExecution), s.handleTaskExecution)
	s.mux.HandleFunc(string(JobTypeRecurringTask), s.handleRecurringTask)
	s.mux.HandleFunc(string(JobTypeTaskReminder), s.handleTaskReminder)
	s.mux.HandleFunc(string(JobTypeTaskDeadline), s.handleTaskDeadline)
}

// ScheduleTask schedules a task execution
func (s *AsynqSchedulerService) ScheduleTask(ctx context.Context, req *TaskExecutionRequest) error {
	payload := &JobPayload{
		JobType:   req.JobType,
		TaskID:    req.Task.ID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		ExecuteAt: req.ExecuteAt,
		Metadata:  req.Metadata,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	task := asynq.NewTask(string(req.JobType), payloadBytes)

	// Calculate delay from now to execution time
	delay := time.Until(req.ExecuteAt)

	info, err := s.client.Enqueue(task, asynq.ProcessIn(delay))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	s.logger.Info(fmt.Sprintf("Scheduled task %s for execution at %s (queue: %s, id: %s)",
		req.Task.ID, req.ExecuteAt.Format(time.RFC3339), info.Queue, info.ID))

	return nil
}

// ScheduleTaskExecution schedules immediate task execution
func (s *AsynqSchedulerService) ScheduleTaskExecution(ctx context.Context, taskID, userID uuid.UUID, executeAt time.Time) error {
	// Fetch task details using correct method signature
	taskResp, err := s.taskService.GetTask(ctx, userID.String(), taskID.String())
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Convert to domain task
	domainTask := s.convertResponseToTask(taskResp)

	req := &TaskExecutionRequest{
		Task:      domainTask,
		UserID:    userID,
		ExecuteAt: executeAt,
		JobType:   JobTypeTaskExecution,
	}

	return s.ScheduleTask(ctx, req)
}

// ScheduleTaskReminder schedules a task reminder
func (s *AsynqSchedulerService) ScheduleTaskReminder(ctx context.Context, taskID, userID uuid.UUID, remindAt time.Time) error {
	taskResp, err := s.taskService.GetTask(ctx, userID.String(), taskID.String())
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	domainTask := s.convertResponseToTask(taskResp)

	req := &TaskExecutionRequest{
		Task:      domainTask,
		UserID:    userID,
		ExecuteAt: remindAt,
		JobType:   JobTypeTaskReminder,
	}

	return s.ScheduleTask(ctx, req)
}

// ScheduleRecurringTask schedules the next occurrence of a recurring task
func (s *AsynqSchedulerService) ScheduleRecurringTask(ctx context.Context, taskID, userID uuid.UUID, nextRun time.Time) error {
	taskResp, err := s.taskService.GetTask(ctx, userID.String(), taskID.String())
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	domainTask := s.convertResponseToTask(taskResp)

	req := &TaskExecutionRequest{
		Task:      domainTask,
		UserID:    userID,
		ExecuteAt: nextRun,
		JobType:   JobTypeRecurringTask,
	}

	return s.ScheduleTask(ctx, req)
}

// CancelScheduledTask cancels a scheduled task (placeholder - asynq doesn't support easy cancellation)
func (s *AsynqSchedulerService) CancelScheduledTask(ctx context.Context, taskID uuid.UUID) error {
	// Note: asynq doesn't provide easy job cancellation by custom ID
	// This would require maintaining a mapping of task IDs to asynq job IDs
	s.logger.Warn(fmt.Sprintf("Cancel request for task %s - not fully implemented", taskID))
	return nil
}

// RescheduleTask reschedules a task (placeholder implementation)
func (s *AsynqSchedulerService) RescheduleTask(ctx context.Context, taskID uuid.UUID, newTime time.Time) error {
	// This would require canceling the old job and creating a new one
	s.logger.Warn(fmt.Sprintf("Reschedule request for task %s to %s - not fully implemented", taskID, newTime))
	return nil
}

// GetScheduledJobs returns scheduled jobs for a user (placeholder implementation)
func (s *AsynqSchedulerService) GetScheduledJobs(ctx context.Context, userID uuid.UUID) ([]JobPayload, error) {
	// This would require maintaining additional metadata about jobs
	s.logger.Warn(fmt.Sprintf("GetScheduledJobs request for user %s - not fully implemented", userID))
	return []JobPayload{}, nil
}

// Start starts the scheduler server
func (s *AsynqSchedulerService) Start(ctx context.Context) error {
	s.logger.Info("Starting asynq scheduler server...")

	go func() {
		if err := s.server.Run(s.mux); err != nil {
			s.logger.Error(fmt.Sprintf("Asynq server error: %v", err))
		}
	}()

	s.logger.Info("Asynq scheduler server started successfully")
	return nil
}

// Stop stops the scheduler server
func (s *AsynqSchedulerService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping asynq scheduler server...")

	s.server.Shutdown()
	s.client.Close()

	s.logger.Info("Asynq scheduler server stopped")
	return nil
}

// Health checks the scheduler health
func (s *AsynqSchedulerService) Health(ctx context.Context) error {
	// Simple health check - we'll just return nil for now
	// In a production environment, you might want to check Redis connectivity
	return nil
}

// Job Handlers

// handleTaskExecution handles task execution jobs
func (s *AsynqSchedulerService) handleTaskExecution(ctx context.Context, t *asynq.Task) error {
	var payload JobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task execution payload: %w", err)
	}

	result, err := s.processTaskExecution(ctx, &payload)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Task execution failed: %v", err))
		return err
	}

	s.logger.Info(fmt.Sprintf("Task execution completed: %s", result.TaskID))
	return nil
}

// handleRecurringTask handles recurring task jobs
func (s *AsynqSchedulerService) handleRecurringTask(ctx context.Context, t *asynq.Task) error {
	var payload JobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal recurring task payload: %w", err)
	}

	result, err := s.processRecurringTask(ctx, &payload)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Recurring task processing failed: %v", err))
		return err
	}

	// Schedule next occurrence if applicable
	if result.NextRun != nil {
		err = s.ScheduleRecurringTask(ctx, payload.TaskID, payload.UserID, *result.NextRun)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to schedule next occurrence: %v", err))
		}
	}

	s.logger.Info(fmt.Sprintf("Recurring task processed: %s", result.TaskID))
	return nil
}

// handleTaskReminder handles task reminder jobs
func (s *AsynqSchedulerService) handleTaskReminder(ctx context.Context, t *asynq.Task) error {
	var payload JobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task reminder payload: %w", err)
	}

	result, err := s.processTaskReminder(ctx, &payload)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Task reminder failed: %v", err))
		return err
	}

	s.logger.Info(fmt.Sprintf("Task reminder sent: %s", result.TaskID))
	return nil
}

// handleTaskDeadline handles task deadline jobs
func (s *AsynqSchedulerService) handleTaskDeadline(ctx context.Context, t *asynq.Task) error {
	var payload JobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task deadline payload: %w", err)
	}

	// Process as a reminder but mark as deadline
	payload.Metadata = map[string]interface{}{
		"type": "deadline",
	}

	result, err := s.processTaskReminder(ctx, &payload)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Task deadline notification failed: %v", err))
		return err
	}

	s.logger.Info(fmt.Sprintf("Task deadline notification sent: %s", result.TaskID))
	return nil
}

// Processing methods

// processTaskExecution processes task execution jobs
func (s *AsynqSchedulerService) processTaskExecution(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error) {
	// Get task details
	taskResp, err := s.taskService.GetTask(ctx, payload.UserID.String(), payload.TaskID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Convert to domain task
	domainTask := s.convertResponseToTask(taskResp)

	// Create a dedicated brain system for this user's task execution
	userBrainSystem := s.brainSystemFactory.CreateBrainSystem()
	brainIntegration := NewBrainIntegration(userBrainSystem, s.deviceRegistry, s.logger)

	// Execute task using brain system
	result, err := brainIntegration.ExecuteTaskWithBrain(ctx, domainTask, payload.UserID, payload.SessionID, payload.JobType)
	if err != nil {
		return &TaskExecutionResult{
			TaskID:     payload.TaskID,
			UserID:     payload.UserID,
			ExecutedAt: time.Now(),
			Success:    false,
			Error:      err.Error(),
		}, err
	}

	// Mark task as completed if it was successfully executed
	if payload.JobType == JobTypeTaskExecution {
		_, err = s.taskService.MarkTaskCompleted(ctx, payload.UserID.String(), payload.TaskID.String())
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to mark task as completed: %v", err))
		}
	}

	return &TaskExecutionResult{
		TaskID:     payload.TaskID,
		UserID:     payload.UserID,
		ExecutedAt: time.Now(),
		Success:    true,
		Message:    result,
	}, nil
}

// processRecurringTask processes recurring task jobs
func (s *AsynqSchedulerService) processRecurringTask(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error) {
	// Get task details
	taskResp, err := s.taskService.GetTask(ctx, payload.UserID.String(), payload.TaskID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Convert to domain task
	domainTask := s.convertResponseToTask(taskResp)

	// Create a dedicated brain system for this user's task execution
	userBrainSystem := s.brainSystemFactory.CreateBrainSystem()
	brainIntegration := NewBrainIntegration(userBrainSystem, s.deviceRegistry, s.logger)

	// Execute task using brain system
	result, err := brainIntegration.ExecuteTaskWithBrain(ctx, domainTask, payload.UserID, payload.SessionID, payload.JobType)
	if err != nil {
		return &TaskExecutionResult{
			TaskID:     payload.TaskID,
			UserID:     payload.UserID,
			ExecutedAt: time.Now(),
			Success:    false,
			Error:      err.Error(),
		}, err
	}

	// Calculate next execution time for recurring task
	var nextRun *time.Time
	if domainTask.RecurrenceConfig != nil {
		nextExecution := s.calculateNextExecution(domainTask)
		nextRun = &nextExecution
	}

	return &TaskExecutionResult{
		TaskID:     payload.TaskID,
		UserID:     payload.UserID,
		ExecutedAt: time.Now(),
		Success:    true,
		Message:    result,
		NextRun:    nextRun,
	}, nil
}

// processTaskReminder processes task reminder jobs
func (s *AsynqSchedulerService) processTaskReminder(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error) {
	// Get task details
	taskResp, err := s.taskService.GetTask(ctx, payload.UserID.String(), payload.TaskID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Convert to domain task
	domainTask := s.convertResponseToTask(taskResp)

	// Create a dedicated brain system for this user's task execution
	userBrainSystem := s.brainSystemFactory.CreateBrainSystem()
	brainIntegration := NewBrainIntegration(userBrainSystem, s.deviceRegistry, s.logger)

	// Send reminder using brain system
	result, err := brainIntegration.ExecuteTaskWithBrain(ctx, domainTask, payload.UserID, payload.SessionID, payload.JobType)
	if err != nil {
		return &TaskExecutionResult{
			TaskID:     payload.TaskID,
			UserID:     payload.UserID,
			ExecutedAt: time.Now(),
			Success:    false,
			Error:      err.Error(),
		}, err
	}

	return &TaskExecutionResult{
		TaskID:     payload.TaskID,
		UserID:     payload.UserID,
		ExecutedAt: time.Now(),
		Success:    true,
		Message:    result,
	}, nil
}

// Helper methods

// convertResponseToTask converts TaskResponse to domain Task
func (s *AsynqSchedulerService) convertResponseToTask(resp *task.TaskResponse) *task.Task {
	return &task.Task{
		ID:               resp.ID,
		UserID:           resp.UserID,
		Title:            resp.Title,
		Description:      resp.Description,
		Status:           task.TaskStatus(resp.Status),
		Priority:         resp.Priority,
		Tags:             resp.Tags,
		ScheduledAt:      resp.ScheduledAt,
		DueAt:            resp.DueAt,
		CompletedAt:      resp.CompletedAt,
		CancelledAt:      resp.CancelledAt,
		IsRecurring:      resp.IsRecurring,
		RecurrenceConfig: resp.RecurrenceConfig,
		ParentTaskID:     resp.ParentTaskID,
		NextExecution:    resp.NextExecution,
		ExecutionCount:   resp.ExecutionCount,
		Metadata:         resp.Metadata,
		CreatedAt:        resp.CreatedAt,
		UpdatedAt:        resp.UpdatedAt,
	}
}

// calculateNextExecution calculates the next execution time for a recurring task
func (s *AsynqSchedulerService) calculateNextExecution(t *task.Task) time.Time {
	if t.RecurrenceConfig == nil {
		return time.Now()
	}

	now := time.Now()
	config := t.RecurrenceConfig

	switch config.Type {
	case task.RecurrenceDaily:
		return now.Add(time.Duration(config.Interval) * 24 * time.Hour)
	case task.RecurrenceWeekly:
		return now.Add(time.Duration(config.Interval) * 7 * 24 * time.Hour)
	case task.RecurrenceMonthly:
		return now.AddDate(0, config.Interval, 0)
	case task.RecurrenceYearly:
		return now.AddDate(config.Interval, 0, 0)
	default:
		// Default to daily for custom/unknown types
		return now.Add(24 * time.Hour)
	}
}
