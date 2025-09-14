package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/constants/prompts"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/runtime/brain"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
)

// BrainIntegration implements BrainSystemIntegration
type BrainIntegration struct {
	brainSystem    *brain.BrainSystem
	deviceRegistry registry.DeviceRegistry
	logger         *Logger.Logger
}

// NewBrainIntegration creates a new brain integration
func NewBrainIntegration(
	brainSystem *brain.BrainSystem,
	deviceRegistry registry.DeviceRegistry,
	logger *Logger.Logger,
) BrainSystemIntegration {
	return &BrainIntegration{
		brainSystem:    brainSystem,
		deviceRegistry: deviceRegistry,
		logger:         logger,
	}
}

// ExecuteTaskWithBrain executes a task using the brain system with combined prompts
func (bi *BrainIntegration) ExecuteTaskWithBrain(
	ctx context.Context,
	task *task.Task,
	userID, sessionID uuid.UUID,
	jobType JobType,
) (*types.Message, error) {
	
	// Create system messages with combined prompts
	systemMessages := bi.createSystemMessages(task, jobType)
	
	// Create task execution message
	var executionTime time.Time
	if task.DueAt != nil {
		executionTime = *task.DueAt
	} else {
		executionTime = time.Now()
	}
	
	taskMessage := types.Message{
		Id:        uuid.New(),
		UserId:    userID,
		Text:      bi.createTaskExecutionMessage(task, jobType),
		Timestamp: executionTime,
		MsgRole:   assistant.USER,
		Tags:      []string{"task_execution", string(jobType)},
	}
	
	// Combine all messages
	messages := append(systemMessages, taskMessage)
	
	// Process through brain system
	result, err := bi.brainSystem.ProcessMessage(ctx, userID, sessionID, messages)
	if err != nil {
		return nil, fmt.Errorf("brain system task execution failed: %w", err)
	}
	
	bi.logger.Info(fmt.Sprintf("Task %s executed successfully via brain system", task.ID))
	return result, nil
}

// SendTaskNotification sends a task notification to the user through the brain system
func (bi *BrainIntegration) SendTaskNotification(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	message string,
	taskID uuid.UUID,
) error {
	
	// Create notification message
	notificationMessage := types.Message{
		Id:        uuid.New(),
		UserId:    userID,
		Text:      message,
		Timestamp: time.Now(),
		MsgRole:   assistant.ASSISTANT,
		Tags:      []string{"task_notification"},
	}
	
	// Send through brain system with streaming to push to connected devices
	err := bi.brainSystem.ProcessMessageWithStreaming(ctx, userID, sessionID, []types.Message{notificationMessage}, false)
	if err != nil {
		return fmt.Errorf("failed to send task notification: %w", err)
	}
	
	bi.logger.Info(fmt.Sprintf("Task notification sent for task %s to user %s", taskID, userID))
	return nil
}

// createSystemMessages creates system messages with combined default and task prompts
func (bi *BrainIntegration) createSystemMessages(task *task.Task, jobType JobType) []types.Message {
	var systemMessages []types.Message
	
	// Add default system prompt
	defaultPrompt := prompts.DEFAULT_PROMPT.Items[prompts.DEFAULT_PROMPT.CurrentVersion]
	systemMessages = append(systemMessages, types.Message{
		Id:        uuid.New(),
		UserId:    uuid.Nil, // System message
		Text:      defaultPrompt.Content,
		Timestamp: time.Now(),
		MsgRole:   assistant.SYSTEM,
		Tags:      []string{"system_prompt", "default"},
	})
	
	// Add task-specific system prompt
	taskPrompt := prompts.TASK_PROMPT.Items[prompts.TASK_PROMPT.CurrentVersion]
	systemMessages = append(systemMessages, types.Message{
		Id:        uuid.New(),
		UserId:    uuid.Nil, // System message
		Text:      taskPrompt.Content,
		Timestamp: time.Now(),
		MsgRole:   assistant.SYSTEM,
		Tags:      []string{"system_prompt", "task", string(jobType)},
	})
	
	return systemMessages
}

// createTaskExecutionMessage creates the task execution message for the brain system
func (bi *BrainIntegration) createTaskExecutionMessage(task *task.Task, jobType JobType) string {
	var dueDate string
	if task.DueAt != nil {
		dueDate = task.DueAt.Format("2006-01-02 15:04:05")
	} else {
		dueDate = "No deadline set"
	}
	
	switch jobType {
	case JobTypeTaskExecution:
		return fmt.Sprintf("Execute the task: %s\n\nDescription: %s\n\nDue: %s\n\nStatus: %s",
			task.Title,
			task.Description,
			dueDate,
			task.Status,
		)
	case JobTypeTaskReminder:
		return fmt.Sprintf("Remind the user about the task: %s\n\nDescription: %s\n\nDue: %s\n\nThis is a friendly reminder about this upcoming task.",
			task.Title,
			task.Description,
			dueDate,
		)
	case JobTypeTaskDeadline:
		return fmt.Sprintf("DEADLINE ALERT: The task '%s' is due now!\n\nDescription: %s\n\nDue: %s\n\nThis task requires immediate attention.",
			task.Title,
			task.Description,
			dueDate,
		)
	case JobTypeRecurringTask:
		var recurrenceInfo string
		if task.RecurrenceConfig != nil {
			recurrenceInfo = string(task.RecurrenceConfig.Type)
		} else {
			recurrenceInfo = "No recurrence configured"
		}
		return fmt.Sprintf("Execute recurring task: %s\n\nDescription: %s\n\nRecurrence: %s\n\nThis is a recurring task that should be executed now.",
			task.Title,
			task.Description,
			recurrenceInfo,
		)
	default:
		return fmt.Sprintf("Process task: %s\n\nDescription: %s", task.Title, task.Description)
	}
}
