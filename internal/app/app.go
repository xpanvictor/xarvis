package app

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-redis/redis"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/note"
	"github.com/xpanvictor/xarvis/internal/domains/project"
	"github.com/xpanvictor/xarvis/internal/domains/scheduler"
	sys_manager "github.com/xpanvictor/xarvis/internal/domains/sys_manager"
	"github.com/xpanvictor/xarvis/internal/domains/task"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/internal/models/processor"
	convoRepo "github.com/xpanvictor/xarvis/internal/repository/conversation"
	noteRepo "github.com/xpanvictor/xarvis/internal/repository/note"
	projectRepo "github.com/xpanvictor/xarvis/internal/repository/project"
	taskRepo "github.com/xpanvictor/xarvis/internal/repository/task"
	userRepo "github.com/xpanvictor/xarvis/internal/repository/user"
	"github.com/xpanvictor/xarvis/internal/runtime/brain"
	"github.com/xpanvictor/xarvis/internal/runtime/embedding"
	"github.com/xpanvictor/xarvis/internal/server"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	memoryregistry "github.com/xpanvictor/xarvis/pkg/io/registry/memoryRegistry"
	"gorm.io/gorm"
)

// App represents the application with all its dependencies
type App struct {
	Config         *config.Settings
	Logger         *Logger.Logger
	DB             *gorm.DB
	RC             *redis.Client
	DeviceRegistry registry.DeviceRegistry
	LLMRouter      *router.Mux
	Embedder       embedding.Embedder
	// repos
	ConversationRepo conversation.ConversationRepository
	UserRepo         user.UserRepository
	ProjectRepo      project.ProjectRepository
	NoteRepo         note.NoteRepository
	TaskRepo         task.TaskRepository
	// services
	SchedulerService scheduler.SchedulerService
	// system components
	Processor       processor.Processor
	SystemManager   *sys_manager.SystemManager
	ServerDeps      server.Dependencies
}

// NewApp creates a new application instance with all dependencies properly wired
func NewApp(cfg *config.Settings, logger *Logger.Logger, db *gorm.DB, rc *redis.Client) (*App, error) {
	app := &App{
		Config: cfg,
		Logger: logger,
		DB:     db,
		RC:     rc,
	}

	if err := app.setupDependencies(); err != nil {
		return nil, err
	}

	return app, nil
}

// setupDependencies initializes all application dependencies
func (a *App) setupDependencies() error {
	// 1. Create shared device registry
	a.DeviceRegistry = memoryregistry.New()

	// 2. Set up embedder
	if err := a.setupEmbedder(); err != nil {
		return err
	}

	//  3. Set up LLM providers and router
	if err := a.setupLLMRouter(); err != nil {
		return err
	}

	// 4. setup deps
	deps := server.NewServerDependencies(
		a.ConversationRepo,
		a.DeviceRegistry,
		a.LLMRouter,
		a.Config.BrainConfig,
		a.Logger,
		a.Config,
	)
	// 5. Set up repositories
	a.ConversationRepo = convoRepo.NewGormConvoRepo(a.DB, a.RC, time.Duration(a.Config.BrainConfig.MsgTTLMins*int64(time.Minute)), a.Embedder)
	a.UserRepo = userRepo.NewGormUserRepo(a.DB)
	a.ProjectRepo = projectRepo.NewGormProjectRepo(a.DB)
	a.NoteRepo = noteRepo.NewGormNoteRepo(a.DB)
	a.TaskRepo = taskRepo.NewGormTaskRepo(a.DB)

	// Set up processor if enabled
	if err := a.setupProcessor(); err != nil {
		return err
	}

	// Set up scheduler service
	schedulerConfig := scheduler.AsynqSchedulerConfig{
		RedisAddr:     "localhost:6379", // Use default Redis address for now
		RedisPassword: "",               // No password for local Redis
		RedisDB:       0,                // Default Redis DB
		Concurrency:   10,
		Queues: map[string]int{
			"default": 6,
			"high":    3,
			"low":     1,
		},
	}

	// Create brain system for scheduler similar to conversation service
	piperURL, err := url.Parse(a.Config.Voice.TTSURL)
	if err != nil {
		return fmt.Errorf("failed to parse TTS URL: %w", err)
	}
	
	schedulerBrainSystem := brain.NewBrainSystem(
		a.Config.BrainConfig,
		a.LLMRouter,
		a.DeviceRegistry,
		piperURL,
		a.Logger,
	)

	// Create scheduler with brain system
	a.SchedulerService = scheduler.NewAsynqSchedulerService(
		schedulerConfig,
		a.Logger,
		nil, // TaskService will be set later to avoid circular dependency
		a.DeviceRegistry,
		a.LLMRouter,
		schedulerBrainSystem,
	)
	a.TaskRepo = taskRepo.NewGormTaskRepo(a.DB)

	// JWT settings from config
	jwtSecret := a.Config.Auth.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
		a.Logger.Warn("JWT secret not configured, using default (not secure for production)")
	}

	tokenTTLHours := a.Config.Auth.TokenTTLHours
	if tokenTTLHours == 0 {
		tokenTTLHours = 24 // default 24 hours
	}
	tokenTTL := time.Duration(tokenTTLHours) * time.Hour

	// add services
	deps.UserService = user.NewUserService(a.UserRepo, a.Logger, jwtSecret, tokenTTL)
	deps.ProjectService = project.NewProjectService(a.ProjectRepo, a.Logger)
	deps.NoteService = note.NewNoteService(a.NoteRepo, a.Logger)
	
	// Create task service and wire with scheduler
	taskService := task.NewTaskService(a.TaskRepo, a.Logger)
	deps.TaskService = taskService
	
	// Set up circular dependency between task service and scheduler
	schedulerAdapter := scheduler.NewTaskSchedulerAdapter(a.SchedulerService)
	if taskServiceImpl, ok := taskService.(interface{ SetScheduler(task.TaskScheduler) }); ok {
		taskServiceImpl.SetScheduler(schedulerAdapter)
	}
	
	deps.ConversationService = conversation.New(
		*deps.Configs,
		deps.Mux,
		deps.DeviceRegistry,
		deps.Logger,
		a.ConversationRepo,
	) // Create conversation service
	
	// Start the scheduler
	go func() {
		if err := a.SchedulerService.Start(context.Background()); err != nil {
			a.Logger.Error("Failed to start scheduler:", err)
		}
	}()

	a.ServerDeps = deps

	// Set up system manager after all services are initialized
	if err := a.setupSystemManager(); err != nil {
		return err
	}

	return nil
}

// setupEmbedder configures the embedder
func (a *App) setupEmbedder() error {
	// For now, hardcode TEI URL from docker-compose
	// In production, this should come from config
	teiURL := "http://embeddings-tei:80"

	// Create TEI embedder
	a.Embedder = embedding.NewTEIEmbedder(teiURL, a.Logger)

	a.Logger.Info("TEI embedder configured successfully")
	return nil
}

// setupLLMRouter configures the LLM providers and creates the router
func (a *App) setupLLMRouter() error {
	// Create LLM router factory with configuration from settings
	factory := NewLLMRouterFactory(a.Config, a.Logger)

	// Create the router
	mux, err := factory.CreateRouter()
	if err != nil {
		return err
	}

	a.LLMRouter = mux
	return nil
}

// GetServerDependencies returns the server dependencies
func (a *App) GetServerDependencies() server.Dependencies {
	return a.ServerDeps
}

// setupProcessor initializes the processor for system decision making
func (a *App) setupProcessor() error {
	if !a.Config.Processor.Enabled {
		a.Logger.Info("Processor disabled in configuration")
		return nil
	}
	
	// Set up Gemini processor
	geminiConfig := processor.GeminiConfig{
		APIKey:    a.Config.Processor.GeminiAPIKey,
		ModelName: a.Config.Processor.GeminiModel,
	}
	
	if geminiConfig.APIKey == "" {
		a.Logger.Warn("Gemini API key not configured, processor will not be available")
		return nil
	}
	
	geminiProcessor, err := processor.NewGeminiProcessor(geminiConfig, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to create Gemini processor: %w", err)
	}
	
	a.Processor = geminiProcessor
	a.Logger.Info("Gemini processor configured successfully")
	
	return nil
}

// setupSystemManager initializes the system manager and background tasks
func (a *App) setupSystemManager() error {
	if !a.Config.SystemManager.Enabled {
		a.Logger.Info("System manager disabled in configuration")
		return nil
	}
	
	// Create system manager
	a.SystemManager = sys_manager.NewSystemManager(a.Logger)
	
	// Set up message summarizer task if processor is available
	if a.Processor != nil {
		// Parse interval from config
		interval := 3 * time.Minute // default
		if a.Config.SystemManager.MessageSummarizerInterval != "" {
			parsedInterval, err := time.ParseDuration(a.Config.SystemManager.MessageSummarizerInterval)
			if err != nil {
				a.Logger.Warn(fmt.Sprintf("Invalid message summarizer interval '%s', using default 3m", 
					a.Config.SystemManager.MessageSummarizerInterval))
			} else {
				interval = parsedInterval
			}
		}
		
		// Create message summarizer
		summarizer := convoRepo.NewMessageSummarizer(a.Processor, a.Logger, a.ConversationRepo)
		
		// Create and register the summarizer task
		summarizerTask := sys_manager.NewMessageSummarizerTask(
			summarizer,
			a.ConversationRepo,
			a.Logger,
			interval,
		)
		
		a.SystemManager.RegisterTask(summarizerTask)
		a.Logger.Info(fmt.Sprintf("Message summarizer task registered with interval: %s", interval))
	} else {
		a.Logger.Warn("Processor not available, skipping message summarizer task")
	}
	
	// Start the system manager
	if err := a.SystemManager.Start(); err != nil {
		return fmt.Errorf("failed to start system manager: %w", err)
	}
	
	a.Logger.Info("System manager started successfully")
	return nil
}

// Shutdown gracefully stops all application components
func (a *App) Shutdown(ctx context.Context) error {
	a.Logger.Info("Shutting down application...")
	
	// Stop system manager if running
	if a.SystemManager != nil && a.SystemManager.IsRunning() {
		if err := a.SystemManager.Stop(); err != nil {
			a.Logger.Error(fmt.Sprintf("Error stopping system manager: %v", err))
		}
	}
	
	// Stop scheduler service if running
	if a.SchedulerService != nil {
		if err := a.SchedulerService.Stop(ctx); err != nil {
			a.Logger.Error(fmt.Sprintf("Error stopping scheduler service: %v", err))
		}
	}
	
	// Close processor if it has a close method
	if processor, ok := a.Processor.(interface{ Close() error }); ok && processor != nil {
		if err := processor.Close(); err != nil {
			a.Logger.Error(fmt.Sprintf("Error closing processor: %v", err))
		}
	}
	
	a.Logger.Info("Application shutdown complete")
	return nil
}
