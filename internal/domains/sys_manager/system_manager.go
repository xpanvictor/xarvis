package sys_manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/repository/conversation"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// SystemTask represents a background task that can be executed
type SystemTask interface {
	// Execute runs the task
	Execute(ctx context.Context) error
	// GetName returns the task name for logging
	GetName() string
	// GetInterval returns how often this task should run
	GetInterval() time.Duration
}

// SystemManager manages and schedules background system tasks
type SystemManager struct {
	tasks   []SystemTask
	logger  *Logger.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
}

// NewSystemManager creates a new system manager
func NewSystemManager(logger *Logger.Logger) *SystemManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &SystemManager{
		tasks:  make([]SystemTask, 0),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// RegisterTask adds a new task to be managed
func (sm *SystemManager) RegisterTask(task SystemTask) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.tasks = append(sm.tasks, task)
	sm.logger.Info(fmt.Sprintf("Registered system task: %s (interval: %s)",
		task.GetName(), task.GetInterval()))
}

// Start begins executing all registered tasks on their schedules
func (sm *SystemManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("system manager is already running")
	}

	sm.running = true
	sm.logger.Info(fmt.Sprintf("Starting system manager with %d tasks", len(sm.tasks)))

	// Start each task in its own goroutine with its own ticker
	for _, task := range sm.tasks {
		sm.wg.Add(1)
		go sm.runTask(task)
	}

	return nil
}

// Stop gracefully shuts down all tasks
func (sm *SystemManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	sm.logger.Info("Stopping system manager...")
	sm.cancel()
	sm.wg.Wait()
	sm.running = false
	sm.logger.Info("System manager stopped")

	return nil
}

// IsRunning returns whether the system manager is currently running
func (sm *SystemManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// GetTaskCount returns the number of registered tasks
func (sm *SystemManager) GetTaskCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.tasks)
}

// runTask executes a single task on its schedule
func (sm *SystemManager) runTask(task SystemTask) {
	defer sm.wg.Done()

	taskName := task.GetName()
	interval := task.GetInterval()

	sm.logger.Info(fmt.Sprintf("Starting task scheduler for: %s", taskName))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Execute immediately on start
	sm.executeTask(task)

	// Then execute on schedule
	for {
		select {
		case <-sm.ctx.Done():
			sm.logger.Info(fmt.Sprintf("Task scheduler stopping for: %s", taskName))
			return
		case <-ticker.C:
			sm.executeTask(task)
		}
	}
}

// executeTask safely executes a task with error handling and logging
func (sm *SystemManager) executeTask(task SystemTask) {
	taskName := task.GetName()
	start := time.Now()

	sm.logger.Debug(fmt.Sprintf("Executing system task: %s", taskName))

	// Create a timeout context for the task
	taskCtx, cancel := context.WithTimeout(sm.ctx, 30*time.Second)
	defer cancel()

	err := task.Execute(taskCtx)
	duration := time.Since(start)

	if err != nil {
		sm.logger.Error(fmt.Sprintf("System task %s failed after %s: %v", taskName, duration, err))
	} else {
		sm.logger.Debug(fmt.Sprintf("System task %s completed in %s", taskName, duration))
	}
}

// MessageSummarizerTask implements SystemTask for message to memory conversion
type MessageSummarizerTask struct {
	summarizer     *conversation.MessageSummarizer
	userRepository types.ConversationRepository
	logger         *Logger.Logger
	interval       time.Duration
}

// NewMessageSummarizerTask creates a new message summarizer task
func NewMessageSummarizerTask(
	summarizer *conversation.MessageSummarizer,
	userRepo types.ConversationRepository,
	logger *Logger.Logger,
	interval time.Duration,
) *MessageSummarizerTask {
	if interval == 0 {
		interval = 3 * time.Minute // Default to 3 minutes
	}

	return &MessageSummarizerTask{
		summarizer:     summarizer,
		userRepository: userRepo,
		logger:         logger,
		interval:       interval,
	}
}

// Execute implements SystemTask.Execute
func (mst *MessageSummarizerTask) Execute(ctx context.Context) error {
	// TODO: We need a way to get active users from the system
	// For now, this is a placeholder that would need to be connected
	// to user management or active session tracking

	activeUsers, err := mst.getActiveUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	if len(activeUsers) == 0 {
		mst.logger.Debug("No active users found for message summarization")
		return nil
	}

	mst.logger.Debug(fmt.Sprintf("Processing message summarization for %d active users", len(activeUsers)))

	// Process each user's messages
	for _, userID := range activeUsers {
		err := mst.summarizer.ProcessAndStoreMemory(ctx, userID)
		if err != nil {
			mst.logger.Error(fmt.Sprintf("Failed to process memories for user %s: %v", userID, err))
			// Continue with other users even if one fails
		}
	}

	return nil
}

// GetName implements SystemTask.GetName
func (mst *MessageSummarizerTask) GetName() string {
	return "MessageSummarizerTask"
}

// GetInterval implements SystemTask.GetInterval
func (mst *MessageSummarizerTask) GetInterval() time.Duration {
	return mst.interval
}

// getActiveUsers retrieves users who have had recent activity
func (mst *MessageSummarizerTask) getActiveUsers(ctx context.Context) ([]uuid.UUID, error) {
	// Use the summarizer to get users with recent messages
	return mst.summarizer.GetActiveUsersWithRecentMessages(ctx, mst.interval)
}

// SystemManagerConfig holds configuration for the system manager
type SystemManagerConfig struct {
	MessageSummarizerInterval time.Duration `json:"message_summarizer_interval"`
	// Add other task intervals here as needed
}

// DefaultSystemManagerConfig returns default configuration
func DefaultSystemManagerConfig() SystemManagerConfig {
	return SystemManagerConfig{
		MessageSummarizerInterval: 3 * time.Minute,
	}
}
