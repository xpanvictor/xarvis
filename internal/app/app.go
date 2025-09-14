package app

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-redis/redis"
	"github.com/xpanvictor/xarvis/internal/app/toolsetup"
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
	"github.com/xpanvictor/xarvis/internal/tools"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	memoryregistry "github.com/xpanvictor/xarvis/pkg/io/registry/memoryRegistry"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
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
	Processor          processor.Processor
	SystemManager      *sys_manager.SystemManager
	BrainSystemFactory *brain.BrainSystemFactory // Factory for creating brain systems with tools
	ServerDeps         server.Dependencies
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

	// 4. setup deps structure
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

	// 6. Create basic services needed for tools
	// JWT settings from config
	var jwtSecret string
	var tokenTTLHours int
	var tokenTTL time.Duration

	jwtSecret = a.Config.Auth.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production"
		a.Logger.Warn("JWT secret not configured, using default (not secure for production)")
	}

	tokenTTLHours = a.Config.Auth.TokenTTLHours
	if tokenTTLHours == 0 {
		tokenTTLHours = 24 // default 24 hours
	}
	tokenTTL = time.Duration(tokenTTLHours) * time.Hour

	// Create services that tools depend on
	deps.UserService = user.NewUserService(a.UserRepo, a.Logger, jwtSecret, tokenTTL)
	deps.ProjectService = project.NewProjectService(a.ProjectRepo, a.Logger)
	deps.NoteService = note.NewNoteService(a.NoteRepo, a.Logger)
	deps.TaskService = task.NewTaskService(a.TaskRepo, a.Logger)

	// Create conversation service early so tools can access it
	deps.ConversationService = conversation.New(
		*deps.Configs,
		deps.Mux,
		deps.DeviceRegistry,
		deps.Logger,
		a.ConversationRepo,
		nil, // BrainSystemFactory will be set later
	)

	// Store deps for tools setup
	a.ServerDeps = deps

	// 7. Set up tools with dependency injection (must be before services that need BrainSystemFactory)
	if err := a.setupTools(); err != nil {
		return err
	}

	// 8. Set up processor if enabled
	if err := a.setupProcessor(); err != nil {
		return err
	}

	// Set up scheduler service
	schedulerConfig := scheduler.AsynqSchedulerConfig{
		RedisAddr:     a.Config.RedisDB.Addr, // Use the same Redis address from config
		RedisPassword: a.Config.RedisDB.Pass, // Use the same Redis password from config
		RedisDB:       0,                     // Default Redis DB
		Concurrency:   10,
		Queues: map[string]int{
			"default": 6,
			"high":    3,
			"low":     1,
		},
	}

	// Create scheduler with brain system factory
	a.SchedulerService = scheduler.NewAsynqSchedulerService(
		schedulerConfig,
		a.Logger,
		nil, // TaskService will be set later to avoid circular dependency
		a.DeviceRegistry,
		a.LLMRouter,
		a.BrainSystemFactory,
	)
	a.TaskRepo = taskRepo.NewGormTaskRepo(a.DB)

	// Set up circular dependency between task service and scheduler
	schedulerAdapter := scheduler.NewTaskSchedulerAdapter(a.SchedulerService)
	if taskServiceImpl, ok := deps.TaskService.(interface{ SetScheduler(task.TaskScheduler) }); ok {
		taskServiceImpl.SetScheduler(schedulerAdapter)
	}

	// Update conversation service with the brain system factory
	if convService, ok := deps.ConversationService.(interface {
		SetBrainSystemFactory(*brain.BrainSystemFactory)
	}); ok {
		convService.SetBrainSystemFactory(a.BrainSystemFactory)
	}

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
	// Check if Gemini API key is configured
	geminiAPIKey := a.Config.AssistantKeys.Gemini.APIKey
	if geminiAPIKey == "" {
		return fmt.Errorf("gemini API key not configured in assistantKeys.gemini.gemini_api_key")
	}

	// Create Gemini embedder
	geminiEmbedder, err := embedding.NewGeminiEmbedder(geminiAPIKey, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to create Gemini embedder: %w", err)
	}

	a.Embedder = geminiEmbedder
	a.Logger.Info("Gemini embedder configured successfully")
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

// setupTools initializes the tool system and creates a brain system factory
func (a *App) setupTools() error {
	a.Logger.Info("Setting up tools and brain system factory...")

	// Create tool dependencies
	toolDeps := &tools.ToolDependencies{
		UserService:         a.ServerDeps.UserService,
		ProjectService:      a.ServerDeps.ProjectService,
		NoteService:         a.ServerDeps.NoteService,
		TaskService:         a.ServerDeps.TaskService,
		ConversationService: a.ServerDeps.ConversationService,
		ConversationRepo:    a.ConversationRepo,
		Logger:              a.Logger,
		TavilyAPIKey:        a.Config.ExternalAPIs.TavilyAPIKey,
	}

	// Create tool factory
	toolFactory, err := tools.RegisterAllTools(toolDeps)
	if err != nil {
		return fmt.Errorf("failed to create tool factory: %w", err)
	}

	// Register tool builders
	if err := toolsetup.RegisterToolBuilders(toolFactory); err != nil {
		return fmt.Errorf("failed to register tool builders: %w", err)
	}

	// Build all tools
	registeredTools, err := toolFactory.BuildAllTools()
	if err != nil {
		return fmt.Errorf("failed to build tools: %w", err)
	}

	// Create a tool registry and register all tools
	masterToolRegistry := toolsystem.NewMemoryRegistry()
	for _, tool := range registeredTools {
		if err := masterToolRegistry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool with master registry: %w", err)
		}
	}

	// Register example tools as well (optional)
	if err := toolsystem.RegisterExampleTools(masterToolRegistry); err != nil {
		a.Logger.Warn("Failed to register example tools: %v", err)
	}

	// Parse piper URL for brain system factory
	piperURL, err := url.Parse(a.Config.Voice.TTSURL)
	if err != nil {
		return fmt.Errorf("failed to parse TTS URL: %w", err)
	}

	// Create brain system factory with pre-configured tools
	a.Logger.Info("Creating brain system factory with %d registered tools", len(registeredTools))
	a.Logger.Info("masterToolRegistry has %d tools", len(masterToolRegistry.List()))

	a.BrainSystemFactory = brain.NewBrainSystemFactory(
		a.Config.BrainConfig,
		a.LLMRouter,
		a.DeviceRegistry,
		piperURL,
		a.Logger,
		masterToolRegistry, // Pre-configured with all tools
	)

	a.Logger.Info("Successfully created brain system factory with %d tools", len(registeredTools))
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
