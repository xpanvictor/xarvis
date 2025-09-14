package task

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"gorm.io/gorm"
)

type GormTaskRepo struct {
	db *gorm.DB
}

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrInvalidTaskData = errors.New("invalid task data")
)

// Create implements task.TaskRepository
func (g *GormTaskRepo) Create(t *task.Task) error {
	entity := NewTaskEntityFromDomain(t)
	if err := g.db.Create(entity).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Update domain object with any changes from database
	*t = *entity.ToDomain()
	return nil
}

// GetByID implements task.TaskRepository
func (g *GormTaskRepo) GetByID(id string) (*task.Task, error) {
	var entity TaskEntity
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task by ID: %w", err)
	}
	return entity.ToDomain(), nil
}

// Update implements task.TaskRepository
func (g *GormTaskRepo) Update(id string, updates task.UpdateTaskRequest) (*task.Task, error) {
	var entity TaskEntity

	// First, get the existing task
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task for update: %w", err)
	}

	// Apply updates only for non-nil fields
	updateMap := make(map[string]interface{})

	if updates.Title != nil {
		if *updates.Title == "" {
			return nil, ErrInvalidTaskData
		}
		updateMap["title"] = *updates.Title
	}
	if updates.Description != nil {
		updateMap["description"] = *updates.Description
	}
	if updates.Priority != nil {
		updateMap["priority"] = *updates.Priority
	}
	if updates.Tags != nil {
		updateMap["tags"] = TagList(*updates.Tags)
	}
	if updates.ScheduledAt != nil {
		updateMap["scheduled_at"] = updates.ScheduledAt
	}
	if updates.DueAt != nil {
		updateMap["due_at"] = updates.DueAt
	}
	if updates.IsRecurring != nil {
		updateMap["is_recurring"] = *updates.IsRecurring
	}
	if updates.RecurrenceConfig != nil {
		rc := RecurrenceConfigJSON(*updates.RecurrenceConfig)
		updateMap["recurrence_config"] = &rc
	}
	if updates.Metadata != nil {
		updateMap["metadata"] = MetadataMap(updates.Metadata)
	}

	// Perform the update
	if len(updateMap) > 0 {
		if err := g.db.Model(&entity).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
	}

	// Return the updated task
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	return entity.ToDomain(), nil
}

// Delete implements task.TaskRepository (soft delete)
func (g *GormTaskRepo) Delete(id string) error {
	result := g.db.Where("id = ?", id).Delete(&TaskEntity{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// GetByUserID implements task.TaskRepository
func (g *GormTaskRepo) GetByUserID(userID string, filters task.ListTasksRequest) ([]task.Task, int64, error) {
	var entities []TaskEntity
	var total int64

	query := g.db.Model(&TaskEntity{}).Where("user_id = ?", userID)

	// Apply filters
	query = g.applyFilters(query, filters)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Apply ordering
	query = g.applyOrdering(query, filters)

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert entities to domain objects
	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, total, nil
}

// GetUserTasksByStatus implements task.TaskRepository
func (g *GormTaskRepo) GetUserTasksByStatus(userID string, status task.TaskStatus, offset, limit int) ([]task.Task, int64, error) {
	filters := task.ListTasksRequest{
		Status: &status,
		Offset: offset,
		Limit:  limit,
	}
	return g.GetByUserID(userID, filters)
}

// GetUserTasksByDateRange implements task.TaskRepository
func (g *GormTaskRepo) GetUserTasksByDateRange(userID string, fromDate, toDate time.Time, offset, limit int) ([]task.Task, int64, error) {
	filters := task.ListTasksRequest{
		FromDate: &fromDate,
		ToDate:   &toDate,
		Offset:   offset,
		Limit:    limit,
	}
	return g.GetByUserID(userID, filters)
}

// GetTasksDueToday implements task.TaskRepository
func (g *GormTaskRepo) GetTasksDueToday(userID string) ([]task.Task, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var entities []TaskEntity
	if err := g.db.Where("user_id = ? AND status = ? AND due_at >= ? AND due_at < ?",
		userID, string(task.StatusPending), startOfDay, endOfDay).
		Order("due_at ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks due today: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// GetOverdueTasks implements task.TaskRepository
func (g *GormTaskRepo) GetOverdueTasks(userID string) ([]task.Task, error) {
	now := time.Now()

	var entities []TaskEntity
	if err := g.db.Where("user_id = ? AND status = ? AND due_at < ?",
		userID, string(task.StatusPending), now).
		Order("due_at ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// GetUpcomingTasks implements task.TaskRepository
func (g *GormTaskRepo) GetUpcomingTasks(userID string, days int) ([]task.Task, error) {
	now := time.Now()
	endDate := now.AddDate(0, 0, days)

	var entities []TaskEntity
	if err := g.db.Where("user_id = ? AND status = ? AND (scheduled_at BETWEEN ? AND ? OR due_at BETWEEN ? AND ?)",
		userID, string(task.StatusPending), now, endDate, now, endDate).
		Order("COALESCE(scheduled_at, due_at) ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get upcoming tasks: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// GetTasksToExecute implements task.TaskRepository
func (g *GormTaskRepo) GetTasksToExecute(beforeTime time.Time, limit int) ([]task.Task, error) {
	var entities []TaskEntity
	query := g.db.Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)",
		string(task.StatusPending), beforeTime).
		Order("COALESCE(scheduled_at, created_at) ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks to execute: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// GetRecurringTasks implements task.TaskRepository
func (g *GormTaskRepo) GetRecurringTasks() ([]task.Task, error) {
	var entities []TaskEntity
	if err := g.db.Where("is_recurring = ? AND status = ?", true, string(task.StatusPending)).
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get recurring tasks: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// GetChildTasks implements task.TaskRepository
func (g *GormTaskRepo) GetChildTasks(parentTaskID string) ([]task.Task, error) {
	var entities []TaskEntity
	if err := g.db.Where("parent_task_id = ?", parentTaskID).
		Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get child tasks: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// CreateRecurringInstance implements task.TaskRepository
func (g *GormTaskRepo) CreateRecurringInstance(parentTask *task.Task, scheduledAt time.Time) (*task.Task, error) {
	// Create a new task instance from the parent
	newTask := &task.Task{
		ID:               uuid.New(),
		UserID:           parentTask.UserID,
		Title:            parentTask.Title,
		Description:      parentTask.Description,
		Status:           task.StatusPending,
		Priority:         parentTask.Priority,
		Tags:             parentTask.Tags,
		ScheduledAt:      &scheduledAt,
		DueAt:            parentTask.DueAt,
		IsRecurring:      false, // Instance is not recurring
		RecurrenceConfig: nil,   // Instance doesn't have recurrence config
		ParentTaskID:     &parentTask.ID,
		ExecutionCount:   0,
		Metadata:         parentTask.Metadata,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := g.Create(newTask); err != nil {
		return nil, fmt.Errorf("failed to create recurring instance: %w", err)
	}

	return newTask, nil
}

// Search implements task.TaskRepository
func (g *GormTaskRepo) Search(userID string, query string, filters task.ListTasksRequest) ([]task.Task, int64, error) {
	filters.Search = query
	return g.GetByUserID(userID, filters)
}

// GetByTags implements task.TaskRepository
func (g *GormTaskRepo) GetByTags(userID string, tags []string, offset, limit int) ([]task.Task, int64, error) {
	filters := task.ListTasksRequest{
		Tags:   tags,
		Offset: offset,
		Limit:  limit,
	}
	return g.GetByUserID(userID, filters)
}

// GetCalendarTasks implements task.TaskRepository
func (g *GormTaskRepo) GetCalendarTasks(userID string, fromDate, toDate time.Time) ([]task.CalendarTaskResponse, error) {
	var entities []TaskEntity
	if err := g.db.Where("user_id = ? AND (scheduled_at BETWEEN ? AND ? OR due_at BETWEEN ? AND ?)",
		userID, fromDate, toDate, fromDate, toDate).
		Order("COALESCE(scheduled_at, due_at) ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get calendar tasks: %w", err)
	}

	responses := make([]task.CalendarTaskResponse, len(entities))
	for i, entity := range entities {
		responses[i] = entity.ToCalendarResponse()
	}

	return responses, nil
}

// List implements task.TaskRepository (admin function)
func (g *GormTaskRepo) List(filters task.ListTasksRequest) ([]task.Task, int64, error) {
	var entities []TaskEntity
	var total int64

	query := g.db.Model(&TaskEntity{})

	// Apply filters
	query = g.applyFilters(query, filters)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Apply ordering
	query = g.applyOrdering(query, filters)

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert entities to domain objects
	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, total, nil
}

// GetTasksByUser implements task.TaskRepository
func (g *GormTaskRepo) GetTasksByUser(userID string) ([]task.Task, error) {
	var entities []TaskEntity
	if err := g.db.Where("user_id = ?", userID).
		Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks by user: %w", err)
	}

	tasks := make([]task.Task, len(entities))
	for i, entity := range entities {
		tasks[i] = *entity.ToDomain()
	}

	return tasks, nil
}

// UpdateStatus implements task.TaskRepository
func (g *GormTaskRepo) UpdateStatus(id string, status task.TaskStatus) (*task.Task, error) {
	var entity TaskEntity

	// First, get the existing task
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task for status update: %w", err)
	}

	updateMap := map[string]interface{}{
		"status": string(status),
	}

	// Set completion/cancellation timestamps
	now := time.Now()
	switch status {
	case task.StatusDone:
		if entity.CompletedAt == nil {
			updateMap["completed_at"] = now
			updateMap["execution_count"] = entity.ExecutionCount + 1
		}
	case task.StatusCancelled:
		if entity.CancelledAt == nil {
			updateMap["cancelled_at"] = now
		}
	case task.StatusPending:
		// Reset completion/cancellation timestamps when moving back to pending
		updateMap["completed_at"] = nil
		updateMap["cancelled_at"] = nil
	}

	// Perform the update
	if err := g.db.Model(&entity).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Return the updated task
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	return entity.ToDomain(), nil
}

// BulkUpdateStatus implements task.TaskRepository
func (g *GormTaskRepo) BulkUpdateStatus(ids []string, status task.TaskStatus) error {
	updateMap := map[string]interface{}{
		"status": string(status),
	}

	now := time.Now()
	switch status {
	case task.StatusDone:
		updateMap["completed_at"] = now
	case task.StatusCancelled:
		updateMap["cancelled_at"] = now
	case task.StatusPending:
		updateMap["completed_at"] = nil
		updateMap["cancelled_at"] = nil
	}

	if err := g.db.Model(&TaskEntity{}).Where("id IN ?", ids).Updates(updateMap).Error; err != nil {
		return fmt.Errorf("failed to bulk update task status: %w", err)
	}

	return nil
}

// Helper method to apply filters to queries
func (g *GormTaskRepo) applyFilters(query *gorm.DB, filters task.ListTasksRequest) *gorm.DB {
	if filters.Status != nil {
		query = query.Where("status = ?", string(*filters.Status))
	}

	if filters.Priority != nil {
		query = query.Where("priority = ?", *filters.Priority)
	}

	if filters.IsRecurring != nil {
		query = query.Where("is_recurring = ?", *filters.IsRecurring)
	}

	if filters.IsOverdue != nil && *filters.IsOverdue {
		now := time.Now()
		query = query.Where("status = ? AND due_at < ?", string(task.StatusPending), now)
	}

	if filters.FromDate != nil {
		query = query.Where("(scheduled_at >= ? OR due_at >= ?)", *filters.FromDate, *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("(scheduled_at <= ? OR due_at <= ?)", *filters.ToDate, *filters.ToDate)
	}

	if filters.Search != "" {
		query = query.Where("(title LIKE ? OR description LIKE ?)",
			"%"+filters.Search+"%", "%"+filters.Search+"%")
	}

	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	return query
}

// Helper method to apply ordering to queries
func (g *GormTaskRepo) applyOrdering(query *gorm.DB, filters task.ListTasksRequest) *gorm.DB {
	orderBy := "created_at"
	if filters.OrderBy != "" {
		switch filters.OrderBy {
		case "scheduledAt", "scheduled_at":
			orderBy = "scheduled_at"
		case "dueAt", "due_at":
			orderBy = "due_at"
		case "priority":
			orderBy = "priority"
		case "createdAt", "created_at":
			orderBy = "created_at"
		case "title":
			orderBy = "title"
		case "status":
			orderBy = "status"
		}
	}

	order := "DESC"
	if filters.Order != "" && strings.ToUpper(filters.Order) == "ASC" {
		order = "ASC"
	}

	return query.Order(fmt.Sprintf("%s %s", orderBy, order))
}

// NewGormTaskRepo creates a new GORM-based task repository
func NewGormTaskRepo(db *gorm.DB) task.TaskRepository {
	return &GormTaskRepo{db: db}
}
