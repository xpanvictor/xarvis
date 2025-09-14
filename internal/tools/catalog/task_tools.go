package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/tools"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// TaskCreateToolBuilder builds a tool to create new tasks
type TaskCreateToolBuilder struct{}

func (t *TaskCreateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("create_task", "1.0.0", "Create a new task for the user with optional scheduling and organization features").
		AddStringParameter("title", "The task title or description", true).
		AddStringParameter("description", "Detailed description of the task", false).
		AddNumberParameter("priority", "Task priority from 1 (low) to 5 (high), default is 3", false).
		AddStringParameter("due_at", "Due date and time in ISO format (e.g., 2024-12-25T10:00:00Z)", false).
		AddStringParameter("scheduled_at", "Scheduled execution time in ISO format", false).
		AddArrayParameter("tags", "Tags for task organization", false).
		AddBooleanParameter("is_recurring", "Whether this is a recurring task", false).
		AddStringParameter("recurrence_type", "Type of recurrence: daily, weekly, monthly, yearly", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			title, ok := args["title"].(string)
			if !ok {
				return nil, fmt.Errorf("title parameter is required and must be a string")
			}

			// Build create request
			createReq := task.CreateTaskRequest{
				Title:    title,
				Priority: 3, // Default priority
			}

			// Set optional description
			if desc, exists := args["description"]; exists {
				if descStr, ok := desc.(string); ok {
					createReq.Description = descStr
				}
			}

			// Set priority if provided
			if priority, exists := args["priority"]; exists {
				if priorityFloat, ok := priority.(float64); ok {
					if priorityFloat >= 1 && priorityFloat <= 5 {
						createReq.Priority = int(priorityFloat)
					}
				}
			}

			// Parse due date if provided
			if dueAtStr, exists := args["due_at"]; exists {
				if dueStr, ok := dueAtStr.(string); ok {
					if dueTime, err := time.Parse(time.RFC3339, dueStr); err == nil {
						createReq.DueAt = &dueTime
					} else {
						return nil, fmt.Errorf("invalid due_at format, use ISO 8601 format (e.g., 2024-12-25T10:00:00Z)")
					}
				}
			}

			// Parse scheduled time if provided
			if scheduledAtStr, exists := args["scheduled_at"]; exists {
				if schedStr, ok := scheduledAtStr.(string); ok {
					if schedTime, err := time.Parse(time.RFC3339, schedStr); err == nil {
						createReq.ScheduledAt = &schedTime
					} else {
						return nil, fmt.Errorf("invalid scheduled_at format, use ISO 8601 format")
					}
				}
			}

			// Set tags if provided
			if tags, exists := args["tags"]; exists {
				if tagArray, ok := tags.([]interface{}); ok {
					var tagStrings []string
					for _, tag := range tagArray {
						if tagStr, ok := tag.(string); ok {
							tagStrings = append(tagStrings, tagStr)
						}
					}
					createReq.Tags = tagStrings
				}
			}

			// Handle recurring task setup
			if isRecurring, exists := args["is_recurring"]; exists {
				if recurring, ok := isRecurring.(bool); ok && recurring {
					createReq.IsRecurring = true

					// Set recurrence config if type is provided
					if recType, exists := args["recurrence_type"]; exists {
						if recTypeStr, ok := recType.(string); ok {
							var recurrenceType task.RecurrenceType
							switch strings.ToLower(recTypeStr) {
							case "daily":
								recurrenceType = task.RecurrenceDaily
							case "weekly":
								recurrenceType = task.RecurrenceWeekly
							case "monthly":
								recurrenceType = task.RecurrenceMonthly
							case "yearly":
								recurrenceType = task.RecurrenceYearly
							default:
								return nil, fmt.Errorf("invalid recurrence_type: %s. Use daily, weekly, monthly, or yearly", recTypeStr)
							}

							createReq.RecurrenceConfig = &task.RecurrenceConfig{
								Type: recurrenceType,
							}
						}
					}
				}
			}

			// Create the task
			taskResp, err := deps.TaskService.CreateTask(ctx, userID, createReq)
			if err != nil {
				return nil, fmt.Errorf("failed to create task: %w", err)
			}

			return map[string]any{
				"task":    taskResp,
				"success": true,
				"message": "Task created successfully",
			}, nil
		}).
		AddTags("task", "create", "todo").
		Build()
}

// TaskListToolBuilder builds a tool to fetch user's tasks
type TaskListToolBuilder struct{}

func (t *TaskListToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("fetch_tasks", "1.0.0", "Fetch tasks for the current user with optional filtering").
		AddStringParameter("status", "Filter by task status: pending, done, cancelled", false).
		AddNumberParameter("limit", "Maximum number of tasks to return", false).
		AddNumberParameter("offset", "Number of tasks to skip for pagination", false).
		AddStringParameter("search", "Search term to filter tasks by title and description", false).
		AddArrayParameter("tags", "Filter by specific tags", false).
		AddBooleanParameter("overdue_only", "Only show overdue tasks", false).
		AddBooleanParameter("due_today", "Only show tasks due today", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			// Handle special filters first
			if dueToday, exists := args["due_today"]; exists {
				if dueTodayBool, ok := dueToday.(bool); ok && dueTodayBool {
					tasks, err := deps.TaskService.GetTasksDueToday(ctx, userID)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch tasks due today: %w", err)
					}
					return map[string]any{
						"tasks":   tasks,
						"total":   len(tasks),
						"filter":  "due_today",
						"message": fmt.Sprintf("Found %d tasks due today", len(tasks)),
					}, nil
				}
			}

			if overdueOnly, exists := args["overdue_only"]; exists {
				if overdueBool, ok := overdueOnly.(bool); ok && overdueBool {
					tasks, err := deps.TaskService.GetOverdueTasks(ctx, userID)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch overdue tasks: %w", err)
					}
					return map[string]any{
						"tasks":   tasks,
						"total":   len(tasks),
						"filter":  "overdue",
						"message": fmt.Sprintf("Found %d overdue tasks", len(tasks)),
					}, nil
				}
			}

			// Set defaults for pagination
			limit := 20
			offset := 0

			if l, exists := args["limit"]; exists {
				if limitFloat, ok := l.(float64); ok {
					limit = int(limitFloat)
				}
			}

			if o, exists := args["offset"]; exists {
				if offsetFloat, ok := o.(float64); ok {
					offset = int(offsetFloat)
				}
			}

			// Handle filtering by tags
			if tags, exists := args["tags"]; exists {
				if tagArray, ok := tags.([]interface{}); ok {
					var tagStrings []string
					for _, tag := range tagArray {
						if tagStr, ok := tag.(string); ok {
							tagStrings = append(tagStrings, tagStr)
						}
					}
					if len(tagStrings) > 0 {
						tasks, total, err := deps.TaskService.GetTasksByTags(ctx, userID, tagStrings, offset, limit)
						if err != nil {
							return nil, fmt.Errorf("failed to fetch tasks by tags: %w", err)
						}
						return map[string]any{
							"tasks":  tasks,
							"total":  total,
							"offset": offset,
							"limit":  limit,
							"filter": fmt.Sprintf("tags: %s", strings.Join(tagStrings, ", ")),
						}, nil
					}
				}
			}

			// Handle filtering by status
			if statusStr, exists := args["status"]; exists {
				if status, ok := statusStr.(string); ok {
					var taskStatus task.TaskStatus
					switch strings.ToLower(status) {
					case "pending":
						taskStatus = task.StatusPending
					case "done":
						taskStatus = task.StatusDone
					case "cancelled":
						taskStatus = task.StatusCancelled
					default:
						return nil, fmt.Errorf("invalid status: %s. Use pending, done, or cancelled", status)
					}

					tasks, total, err := deps.TaskService.GetTasksByStatus(ctx, userID, taskStatus, offset, limit)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch tasks by status: %w", err)
					}
					return map[string]any{
						"tasks":  tasks,
						"total":  total,
						"offset": offset,
						"limit":  limit,
						"filter": fmt.Sprintf("status: %s", status),
					}, nil
				}
			}

			// Create list request
			listReq := task.ListTasksRequest{
				Offset: offset,
				Limit:  limit,
			}

			// Add search if provided
			if search, exists := args["search"]; exists {
				if searchStr, ok := search.(string); ok {
					listReq.Search = searchStr
					// Use search method
					tasks, total, err := deps.TaskService.SearchTasks(ctx, userID, searchStr, listReq)
					if err != nil {
						return nil, fmt.Errorf("failed to search tasks: %w", err)
					}
					return map[string]any{
						"tasks":  tasks,
						"total":  total,
						"offset": offset,
						"limit":  limit,
						"search": searchStr,
					}, nil
				}
			}

			// Default: get all user tasks
			tasks, total, err := deps.TaskService.ListUserTasks(ctx, userID, listReq)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch tasks: %w", err)
			}

			return map[string]any{
				"tasks":  tasks,
				"total":  total,
				"offset": offset,
				"limit":  limit,
			}, nil
		}).
		AddTags("task", "list", "fetch").
		Build()
}

// TaskUpdateToolBuilder builds a tool to update existing tasks
type TaskUpdateToolBuilder struct{}

func (t *TaskUpdateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("update_task", "1.0.0", "Update an existing task's information").
		AddStringParameter("task_id", "The task ID to update", true).
		AddStringParameter("title", "New task title", false).
		AddStringParameter("description", "New task description", false).
		AddStringParameter("status", "New task status: pending, done, cancelled", false).
		AddNumberParameter("priority", "New task priority from 1 to 5", false).
		AddStringParameter("due_at", "New due date in ISO format", false).
		AddArrayParameter("tags", "New task tags", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			taskID, ok := args["task_id"].(string)
			if !ok {
				return nil, fmt.Errorf("task_id parameter is required and must be a string")
			}

			// Handle status update separately if that's all that's requested
			if statusStr, exists := args["status"]; exists {
				if status, ok := statusStr.(string); ok {
					var taskStatus task.TaskStatus
					switch strings.ToLower(status) {
					case "pending":
						taskStatus = task.StatusPending
					case "done":
						taskStatus = task.StatusDone
					case "cancelled":
						taskStatus = task.StatusCancelled
					default:
						return nil, fmt.Errorf("invalid status: %s. Use pending, done, or cancelled", status)
					}

					// Use status-specific update method
					statusReq := task.UpdateTaskStatusRequest{Status: taskStatus}
					taskResp, err := deps.TaskService.UpdateTaskStatus(ctx, userID, taskID, statusReq)
					if err != nil {
						return nil, fmt.Errorf("failed to update task status: %w", err)
					}

					return map[string]any{
						"task":    taskResp,
						"success": true,
						"message": fmt.Sprintf("Task status updated to %s", status),
					}, nil
				}
			}

			// Build update request for other fields
			updateReq := task.UpdateTaskRequest{}

			if title, exists := args["title"]; exists {
				if titleStr, ok := title.(string); ok {
					updateReq.Title = &titleStr
				}
			}

			if description, exists := args["description"]; exists {
				if descStr, ok := description.(string); ok {
					updateReq.Description = &descStr
				}
			}

			if priority, exists := args["priority"]; exists {
				if priorityFloat, ok := priority.(float64); ok {
					if priorityFloat >= 1 && priorityFloat <= 5 {
						priorityInt := int(priorityFloat)
						updateReq.Priority = &priorityInt
					}
				}
			}

			if dueAtStr, exists := args["due_at"]; exists {
				if dueStr, ok := dueAtStr.(string); ok {
					if dueTime, err := time.Parse(time.RFC3339, dueStr); err == nil {
						updateReq.DueAt = &dueTime
					} else {
						return nil, fmt.Errorf("invalid due_at format, use ISO 8601 format")
					}
				}
			}

			if tags, exists := args["tags"]; exists {
				if tagArray, ok := tags.([]interface{}); ok {
					var tagStrings []string
					for _, tag := range tagArray {
						if tagStr, ok := tag.(string); ok {
							tagStrings = append(tagStrings, tagStr)
						}
					}
					updateReq.Tags = &tagStrings
				}
			}

			taskResp, err := deps.TaskService.UpdateTask(ctx, userID, taskID, updateReq)
			if err != nil {
				return nil, fmt.Errorf("failed to update task: %w", err)
			}

			return map[string]any{
				"task":    taskResp,
				"success": true,
				"message": "Task updated successfully",
			}, nil
		}).
		AddTags("task", "update", "modify").
		Build()
}

// TaskCompleteToolBuilder builds a tool to mark tasks as completed
type TaskCompleteToolBuilder struct{}

func (t *TaskCompleteToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("complete_task", "1.0.0", "Mark a task as completed").
		AddStringParameter("task_id", "The task ID to mark as completed", true).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			taskID, ok := args["task_id"].(string)
			if !ok {
				return nil, fmt.Errorf("task_id parameter is required and must be a string")
			}

			taskResp, err := deps.TaskService.MarkTaskCompleted(ctx, userID, taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to complete task: %w", err)
			}

			return map[string]any{
				"task":    taskResp,
				"success": true,
				"message": "Task marked as completed",
			}, nil
		}).
		AddTags("task", "complete", "done").
		Build()
}

// TaskUpcomingToolBuilder builds a tool to get upcoming tasks
type TaskUpcomingToolBuilder struct{}

func (t *TaskUpcomingToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("get_upcoming_tasks", "1.0.0", "Get tasks coming up in the next few days").
		AddNumberParameter("days", "Number of days to look ahead (default: 7)", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			days := 7 // Default to 7 days
			if d, exists := args["days"]; exists {
				if daysFloat, ok := d.(float64); ok {
					days = int(daysFloat)
				}
			}

			tasks, err := deps.TaskService.GetUpcomingTasks(ctx, userID, days)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch upcoming tasks: %w", err)
			}

			return map[string]any{
				"tasks":   tasks,
				"total":   len(tasks),
				"days":    days,
				"message": fmt.Sprintf("Found %d tasks coming up in the next %d days", len(tasks), days),
			}, nil
		}).
		AddTags("task", "upcoming", "schedule").
		Build()
}
