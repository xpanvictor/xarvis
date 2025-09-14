package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TaskSchedulerAdapter implements task.TaskScheduler interface
type TaskSchedulerAdapter struct {
	service SchedulerService
}

// NewTaskSchedulerAdapter creates an adapter for TaskScheduler interface
func NewTaskSchedulerAdapter(service SchedulerService) *TaskSchedulerAdapter {
	return &TaskSchedulerAdapter{service: service}
}

// ScheduleTaskExecution implements task.TaskScheduler.ScheduleTaskExecution
func (a *TaskSchedulerAdapter) ScheduleTaskExecution(ctx context.Context, taskID, userID uuid.UUID, executeAt time.Time) error {
	return a.service.ScheduleTaskExecution(ctx, taskID, userID, executeAt)
}

// ScheduleTaskReminder implements task.TaskScheduler.ScheduleTaskReminder
func (a *TaskSchedulerAdapter) ScheduleTaskReminder(ctx context.Context, taskID, userID uuid.UUID, remindAt time.Time) error {
	return a.service.ScheduleTaskReminder(ctx, taskID, userID, remindAt)
}

// ScheduleRecurringTask implements task.TaskScheduler.ScheduleRecurringTask
func (a *TaskSchedulerAdapter) ScheduleRecurringTask(ctx context.Context, taskID, userID uuid.UUID, nextRun time.Time) error {
	return a.service.ScheduleRecurringTask(ctx, taskID, userID, nextRun)
}
