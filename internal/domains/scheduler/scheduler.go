package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/types"
)

// JobType represents the type of scheduled job
type JobType string

const (
	JobTypeTaskExecution JobType = "task_execution"
	JobTypeRecurringTask JobType = "recurring_task"
	JobTypeTaskReminder  JobType = "task_reminder"
	JobTypeTaskDeadline  JobType = "task_deadline"
)

// JobPayload represents the data structure for scheduled jobs
type JobPayload struct {
	JobType   JobType                `json:"job_type"`
	TaskID    uuid.UUID              `json:"task_id"`
	UserID    uuid.UUID              `json:"user_id"`
	SessionID uuid.UUID              `json:"session_id,omitempty"`
	ExecuteAt time.Time              `json:"execute_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskExecutionRequest represents a request to execute a task
type TaskExecutionRequest struct {
	Task      *task.Task             `json:"task"`
	UserID    uuid.UUID              `json:"user_id"`
	SessionID uuid.UUID              `json:"session_id,omitempty"`
	ExecuteAt time.Time              `json:"execute_at"`
	JobType   JobType                `json:"job_type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskExecutionResult represents the result of task execution
type TaskExecutionResult struct {
	TaskID     uuid.UUID      `json:"task_id"`
	UserID     uuid.UUID      `json:"user_id"`
	ExecutedAt time.Time      `json:"executed_at"`
	Success    bool           `json:"success"`
	Message    *types.Message `json:"message,omitempty"`
	Error      string         `json:"error,omitempty"`
	NextRun    *time.Time     `json:"next_run,omitempty"` // For recurring tasks
}

// SchedulerService defines the interface for task scheduling
type SchedulerService interface {
	// Task scheduling methods
	ScheduleTask(ctx context.Context, req *TaskExecutionRequest) error
	ScheduleTaskExecution(ctx context.Context, taskID, userID uuid.UUID, executeAt time.Time) error
	ScheduleTaskReminder(ctx context.Context, taskID, userID uuid.UUID, remindAt time.Time) error
	ScheduleRecurringTask(ctx context.Context, taskID, userID uuid.UUID, nextRun time.Time) error

	// Job management methods
	CancelScheduledTask(ctx context.Context, taskID uuid.UUID) error
	RescheduleTask(ctx context.Context, taskID uuid.UUID, newTime time.Time) error
	GetScheduledJobs(ctx context.Context, userID uuid.UUID) ([]JobPayload, error)

	// Lifecycle methods
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

// TaskProcessor defines the interface for processing scheduled tasks
type TaskProcessor interface {
	ProcessTaskExecution(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error)
	ProcessTaskReminder(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error)
	ProcessRecurringTask(ctx context.Context, payload *JobPayload) (*TaskExecutionResult, error)
}

// BrainSystemIntegration defines how the scheduler integrates with brain system
type BrainSystemIntegration interface {
	// Execute task using brain system with combined prompts
	ExecuteTaskWithBrain(ctx context.Context, task *task.Task, userID, sessionID uuid.UUID, jobType JobType) (*types.Message, error)
	// Send task notification to user through brain system
	SendTaskNotification(ctx context.Context, userID, sessionID uuid.UUID, message string, taskID uuid.UUID) error
}
