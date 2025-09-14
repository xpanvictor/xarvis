package task

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusCancelled TaskStatus = "cancelled"
	StatusDone      TaskStatus = "done"
)

// IsValid checks if the task status is valid
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusCancelled, StatusDone:
		return true
	default:
		return false
	}
}

// RecurrenceType represents the type of recurrence for tasks
type RecurrenceType string

const (
	RecurrenceNone    RecurrenceType = "none"
	RecurrenceDaily   RecurrenceType = "daily"
	RecurrenceWeekly  RecurrenceType = "weekly"
	RecurrenceMonthly RecurrenceType = "monthly"
	RecurrenceYearly  RecurrenceType = "yearly"
	RecurrenceCustom  RecurrenceType = "custom"
)

// IsValid checks if the recurrence type is valid
func (r RecurrenceType) IsValid() bool {
	switch r {
	case RecurrenceNone, RecurrenceDaily, RecurrenceWeekly, RecurrenceMonthly, RecurrenceYearly, RecurrenceCustom:
		return true
	default:
		return false
	}
}

// RecurrenceConfig represents configuration for recurring tasks
type RecurrenceConfig struct {
	Type           RecurrenceType `json:"type"`
	Interval       int            `json:"interval"`       // For custom recurrence (e.g., every 2 days)
	DaysOfWeek     []int          `json:"daysOfWeek"`     // For weekly (0=Sunday, 1=Monday, etc.)
	DaysOfMonth    []int          `json:"daysOfMonth"`    // For monthly (1-31)
	MonthsOfYear   []int          `json:"monthsOfYear"`   // For yearly (1-12)
	EndDate        *time.Time     `json:"endDate"`        // When to stop recurring
	MaxOccurrences *int           `json:"maxOccurrences"` // Maximum number of occurrences
	TimeZone       string         `json:"timeZone"`       // Timezone for scheduling
}

// Task represents a scheduled task in the system
type Task struct {
	ID               uuid.UUID         `json:"id"`
	UserID           uuid.UUID         `json:"userId"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           TaskStatus        `json:"status"`
	Priority         int               `json:"priority"` // 1=lowest, 5=highest
	Tags             []string          `json:"tags"`
	ScheduledAt      *time.Time        `json:"scheduledAt"`      // When to execute the task
	DueAt            *time.Time        `json:"dueAt"`            // Deadline for the task
	CompletedAt      *time.Time        `json:"completedAt"`      // When task was completed
	CancelledAt      *time.Time        `json:"cancelledAt"`      // When task was cancelled
	IsRecurring      bool              `json:"isRecurring"`      // Whether task is recurring
	RecurrenceConfig *RecurrenceConfig `json:"recurrenceConfig"` // Recurrence configuration
	ParentTaskID     *uuid.UUID        `json:"parentTaskId"`     // For recurring tasks, points to parent
	NextExecution    *time.Time        `json:"nextExecution"`    // For recurring tasks, when is next execution
	ExecutionCount   int               `json:"executionCount"`   // How many times this task has been executed
	Metadata         map[string]any    `json:"metadata"`         // Additional data for brain system
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// NewTask creates a new task
func NewTask(userID uuid.UUID, req CreateTaskRequest) *Task {
	now := time.Now()
	task := &Task{
		ID:               uuid.New(),
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		Status:           StatusPending,
		Priority:         req.Priority,
		Tags:             req.Tags,
		ScheduledAt:      req.ScheduledAt,
		DueAt:            req.DueAt,
		IsRecurring:      req.IsRecurring,
		RecurrenceConfig: req.RecurrenceConfig,
		ExecutionCount:   0,
		Metadata:         req.Metadata,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Set next execution time for recurring tasks
	if task.IsRecurring && task.ScheduledAt != nil {
		task.NextExecution = task.ScheduledAt
	}

	return task
}

// MarkCompleted marks the task as completed
func (t *Task) MarkCompleted() {
	now := time.Now()
	t.Status = StatusDone
	t.CompletedAt = &now
	t.UpdatedAt = now
	t.ExecutionCount++
}

// MarkCancelled marks the task as cancelled
func (t *Task) MarkCancelled() {
	now := time.Now()
	t.Status = StatusCancelled
	t.CancelledAt = &now
	t.UpdatedAt = now
}

// CanExecute checks if the task can be executed now
func (t *Task) CanExecute() bool {
	if t.Status != StatusPending {
		return false
	}

	if t.ScheduledAt == nil {
		return true // Can execute immediately if no schedule
	}

	return time.Now().After(*t.ScheduledAt) || time.Now().Equal(*t.ScheduledAt)
}

// IsOverdue checks if the task is overdue
func (t *Task) IsOverdue() bool {
	if t.Status != StatusPending || t.DueAt == nil {
		return false
	}
	return time.Now().After(*t.DueAt)
}

// ToResponse converts Task to TaskResponse
func (t *Task) ToResponse() TaskResponse {
	return TaskResponse{
		ID:               t.ID,
		UserID:           t.UserID,
		Title:            t.Title,
		Description:      t.Description,
		Status:           t.Status,
		Priority:         t.Priority,
		Tags:             t.Tags,
		ScheduledAt:      t.ScheduledAt,
		DueAt:            t.DueAt,
		CompletedAt:      t.CompletedAt,
		CancelledAt:      t.CancelledAt,
		IsRecurring:      t.IsRecurring,
		RecurrenceConfig: t.RecurrenceConfig,
		ParentTaskID:     t.ParentTaskID,
		NextExecution:    t.NextExecution,
		ExecutionCount:   t.ExecutionCount,
		Metadata:         t.Metadata,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

// CreateTaskRequest represents a request to create a new task
type CreateTaskRequest struct {
	Title            string            `json:"title" binding:"required,min=1,max=200"`
	Description      string            `json:"description" binding:"max=1000"`
	Priority         int               `json:"priority" binding:"min=1,max=5"`
	Tags             []string          `json:"tags"`
	ScheduledAt      *time.Time        `json:"scheduledAt"`
	DueAt            *time.Time        `json:"dueAt"`
	IsRecurring      bool              `json:"isRecurring"`
	RecurrenceConfig *RecurrenceConfig `json:"recurrenceConfig"`
	Metadata         map[string]any    `json:"metadata"`
}

// UpdateTaskRequest represents a request to update a task
type UpdateTaskRequest struct {
	Title            *string           `json:"title" binding:"omitempty,min=1,max=200"`
	Description      *string           `json:"description" binding:"omitempty,max=1000"`
	Priority         *int              `json:"priority" binding:"omitempty,min=1,max=5"`
	Tags             *[]string         `json:"tags"`
	ScheduledAt      *time.Time        `json:"scheduledAt"`
	DueAt            *time.Time        `json:"dueAt"`
	IsRecurring      *bool             `json:"isRecurring"`
	RecurrenceConfig *RecurrenceConfig `json:"recurrenceConfig"`
	Metadata         map[string]any    `json:"metadata"`
}

// UpdateTaskStatusRequest represents a request to update task status
type UpdateTaskStatusRequest struct {
	Status TaskStatus `json:"status" binding:"required"`
}

// ListTasksRequest represents filtering and pagination options for listing tasks
type ListTasksRequest struct {
	Status      *TaskStatus `form:"status"`
	Priority    *int        `form:"priority"`
	Tags        []string    `form:"tags"`
	IsRecurring *bool       `form:"isRecurring"`
	IsOverdue   *bool       `form:"isOverdue"`
	FromDate    *time.Time  `form:"fromDate"`
	ToDate      *time.Time  `form:"toDate"`
	Search      string      `form:"search"`
	OrderBy     string      `form:"orderBy"` // scheduledAt, dueAt, priority, createdAt
	Order       string      `form:"order"`   // asc, desc
	Offset      int         `form:"offset"`
	Limit       int         `form:"limit"`
}

// TaskResponse represents the response format for a task
type TaskResponse struct {
	ID               uuid.UUID         `json:"id"`
	UserID           uuid.UUID         `json:"userId"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           TaskStatus        `json:"status"`
	Priority         int               `json:"priority"`
	Tags             []string          `json:"tags"`
	ScheduledAt      *time.Time        `json:"scheduledAt"`
	DueAt            *time.Time        `json:"dueAt"`
	CompletedAt      *time.Time        `json:"completedAt"`
	CancelledAt      *time.Time        `json:"cancelledAt"`
	IsRecurring      bool              `json:"isRecurring"`
	RecurrenceConfig *RecurrenceConfig `json:"recurrenceConfig"`
	ParentTaskID     *uuid.UUID        `json:"parentTaskId"`
	NextExecution    *time.Time        `json:"nextExecution"`
	ExecutionCount   int               `json:"executionCount"`
	Metadata         map[string]any    `json:"metadata"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// CalendarTaskResponse represents a simplified task view for calendar
type CalendarTaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	ScheduledAt *time.Time `json:"scheduledAt"`
	DueAt       *time.Time `json:"dueAt"`
	IsRecurring bool       `json:"isRecurring"`
}

// TaskRepository defines the interface for task data operations
type TaskRepository interface {
	// Basic CRUD operations
	Create(task *Task) error
	GetByID(id string) (*Task, error)
	Update(id string, updates UpdateTaskRequest) (*Task, error)
	Delete(id string) error

	// User-specific operations
	GetByUserID(userID string, filters ListTasksRequest) ([]Task, int64, error)
	GetUserTasksByStatus(userID string, status TaskStatus, offset, limit int) ([]Task, int64, error)
	GetUserTasksByDateRange(userID string, fromDate, toDate time.Time, offset, limit int) ([]Task, int64, error)

	// Scheduling operations
	GetTasksDueToday(userID string) ([]Task, error)
	GetOverdueTasks(userID string) ([]Task, error)
	GetUpcomingTasks(userID string, days int) ([]Task, error)
	GetTasksToExecute(beforeTime time.Time, limit int) ([]Task, error)

	// Recurring task operations
	GetRecurringTasks() ([]Task, error)
	GetChildTasks(parentTaskID string) ([]Task, error)
	CreateRecurringInstance(parentTask *Task, scheduledAt time.Time) (*Task, error)

	// Search and filtering
	Search(userID string, query string, filters ListTasksRequest) ([]Task, int64, error)
	GetByTags(userID string, tags []string, offset, limit int) ([]Task, int64, error)

	// Calendar operations
	GetCalendarTasks(userID string, fromDate, toDate time.Time) ([]CalendarTaskResponse, error)

	// Admin operations
	List(filters ListTasksRequest) ([]Task, int64, error)
	GetTasksByUser(userID string) ([]Task, error)

	// Status operations
	UpdateStatus(id string, status TaskStatus) (*Task, error)
	BulkUpdateStatus(ids []string, status TaskStatus) error
}
