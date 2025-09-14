package task

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"gorm.io/gorm"
)

// TagList is a custom type for handling JSON serialization of string slices
type TagList []string

// Value implements driver.Valuer interface for GORM
func (t TagList) Value() (driver.Value, error) {
	if len(t) == 0 {
		return "[]", nil
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner interface for GORM
func (t *TagList) Scan(value interface{}) error {
	if value == nil {
		*t = TagList{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		*t = TagList{}
		return nil
	}
}

// MetadataMap is a custom type for handling JSON serialization of metadata
type MetadataMap map[string]any

// Value implements driver.Valuer interface for GORM
func (m MetadataMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner interface for GORM
func (m *MetadataMap) Scan(value interface{}) error {
	if value == nil {
		*m = MetadataMap{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		*m = MetadataMap{}
		return nil
	}
}

// RecurrenceConfigJSON is a custom type for handling JSON serialization of recurrence config
type RecurrenceConfigJSON task.RecurrenceConfig

// Value implements driver.Valuer interface for GORM
func (r RecurrenceConfigJSON) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan implements sql.Scanner interface for GORM
func (r *RecurrenceConfigJSON) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, r)
	case string:
		return json.Unmarshal([]byte(v), r)
	default:
		return nil
	}
}

// TaskEntity represents the database entity for Task with GORM tags
type TaskEntity struct {
	ID               uuid.UUID             `gorm:"primaryKey;type:char(36);not null"`
	UserID           uuid.UUID             `gorm:"column:user_id;type:char(36);not null;index"`
	Title            string                `gorm:"column:title;type:varchar(200);not null"`
	Description      string                `gorm:"column:description;type:text"`
	Status           string                `gorm:"column:status;type:varchar(20);not null;index;default:pending"`
	Priority         int                   `gorm:"column:priority;type:int;not null;default:3"`
	Tags             TagList               `gorm:"type:json;column:tags"`
	ScheduledAt      *time.Time            `gorm:"column:scheduled_at;index"`
	DueAt            *time.Time            `gorm:"column:due_at;index"`
	CompletedAt      *time.Time            `gorm:"column:completed_at"`
	CancelledAt      *time.Time            `gorm:"column:cancelled_at"`
	IsRecurring      bool                  `gorm:"column:is_recurring;default:false;index"`
	RecurrenceConfig *RecurrenceConfigJSON `gorm:"type:json;column:recurrence_config"`
	ParentTaskID     *uuid.UUID            `gorm:"column:parent_task_id;type:char(36);index"`
	NextExecution    *time.Time            `gorm:"column:next_execution;index"`
	ExecutionCount   int                   `gorm:"column:execution_count;default:0"`
	Metadata         MetadataMap           `gorm:"type:json;column:metadata"`
	CreatedAt        time.Time             `gorm:"autoCreateTime(3)"`
	UpdatedAt        time.Time             `gorm:"autoUpdateTime(3)"`
	DeletedAt        *gorm.DeletedAt       `gorm:"index"`
}

// TableName returns the table name for GORM
func (TaskEntity) TableName() string {
	return "tasks"
}

// BeforeCreate is a GORM hook to ensure UUID is set
func (t *TaskEntity) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// ToDomain converts TaskEntity to domain Task
func (t *TaskEntity) ToDomain() *task.Task {
	tags := []string(t.Tags)
	if tags == nil {
		tags = []string{}
	}

	metadata := map[string]any(t.Metadata)
	if metadata == nil {
		metadata = make(map[string]any)
	}

	var recurrenceConfig *task.RecurrenceConfig
	if t.RecurrenceConfig != nil {
		rc := task.RecurrenceConfig(*t.RecurrenceConfig)
		recurrenceConfig = &rc
	}

	return &task.Task{
		ID:               t.ID,
		UserID:           t.UserID,
		Title:            t.Title,
		Description:      t.Description,
		Status:           task.TaskStatus(t.Status),
		Priority:         t.Priority,
		Tags:             tags,
		ScheduledAt:      t.ScheduledAt,
		DueAt:            t.DueAt,
		CompletedAt:      t.CompletedAt,
		CancelledAt:      t.CancelledAt,
		IsRecurring:      t.IsRecurring,
		RecurrenceConfig: recurrenceConfig,
		ParentTaskID:     t.ParentTaskID,
		NextExecution:    t.NextExecution,
		ExecutionCount:   t.ExecutionCount,
		Metadata:         metadata,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

// FromDomain converts domain Task to TaskEntity
func (t *TaskEntity) FromDomain(domainTask *task.Task) {
	t.ID = domainTask.ID
	t.UserID = domainTask.UserID
	t.Title = domainTask.Title
	t.Description = domainTask.Description
	t.Status = string(domainTask.Status)
	t.Priority = domainTask.Priority
	t.Tags = TagList(domainTask.Tags)
	t.ScheduledAt = domainTask.ScheduledAt
	t.DueAt = domainTask.DueAt
	t.CompletedAt = domainTask.CompletedAt
	t.CancelledAt = domainTask.CancelledAt
	t.IsRecurring = domainTask.IsRecurring

	if domainTask.RecurrenceConfig != nil {
		rc := RecurrenceConfigJSON(*domainTask.RecurrenceConfig)
		t.RecurrenceConfig = &rc
	}

	t.ParentTaskID = domainTask.ParentTaskID
	t.NextExecution = domainTask.NextExecution
	t.ExecutionCount = domainTask.ExecutionCount
	t.Metadata = MetadataMap(domainTask.Metadata)
	t.CreatedAt = domainTask.CreatedAt
	t.UpdatedAt = domainTask.UpdatedAt
}

// NewTaskEntityFromDomain creates a new TaskEntity from domain Task
func NewTaskEntityFromDomain(domainTask *task.Task) *TaskEntity {
	entity := &TaskEntity{}
	entity.FromDomain(domainTask)
	return entity
}

// ToCalendarResponse converts TaskEntity to CalendarTaskResponse
func (t *TaskEntity) ToCalendarResponse() task.CalendarTaskResponse {
	return task.CalendarTaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Status:      task.TaskStatus(t.Status),
		Priority:    t.Priority,
		ScheduledAt: t.ScheduledAt,
		DueAt:       t.DueAt,
		IsRecurring: t.IsRecurring,
	}
}
