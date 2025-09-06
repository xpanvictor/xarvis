package app

import (
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/repository"
	"github.com/xpanvictor/xarvis/internal/server"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	memoryregistry "github.com/xpanvictor/xarvis/pkg/io/registry/memoryRegistry"
	"gorm.io/gorm"
)

// App represents the application with all its dependencies
type App struct {
	Config           *config.Settings
	Logger           *Logger.Logger
	DB               *gorm.DB
	DeviceRegistry   registry.DeviceRegistry
	LLMRouter        *router.Mux
	ConversationRepo conversation.ConversationRepository
	ServerDeps       server.Dependencies
}

// NewApp creates a new application instance with all dependencies properly wired
func NewApp(cfg *config.Settings, logger *Logger.Logger, db *gorm.DB) (*App, error) {
	app := &App{
		Config: cfg,
		Logger: logger,
		DB:     db,
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

	// 2. Set up conversation repository
	a.ConversationRepo = repository.NewGormConversationRepo(a.DB)

	// 3. Set up LLM providers and router
	if err := a.setupLLMRouter(); err != nil {
		return err
	}

	// 4. Create server dependencies
	a.ServerDeps = server.NewServerDependencies(
		a.ConversationRepo,
		a.DeviceRegistry,
		a.LLMRouter,
		a.Config.BrainConfig,
		a.Logger,
	)

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
