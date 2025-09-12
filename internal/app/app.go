package app

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	convoRepo "github.com/xpanvictor/xarvis/internal/repository/conversation"
	userRepo "github.com/xpanvictor/xarvis/internal/repository/user"
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
	// repos
	ConversationRepo conversation.ConversationRepository
	UserRepo         user.UserRepository
	ServerDeps       server.Dependencies
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
	//  Set up LLM providers and router
	if err := a.setupLLMRouter(); err != nil {
		return err
	}

	// 2. setup deps
	deps := server.NewServerDependencies(
		a.ConversationRepo,
		a.DeviceRegistry,
		a.LLMRouter,
		a.Config.BrainConfig,
		a.Logger,
		a.Config,
	)
	// 3. Set up repositories
	a.ConversationRepo = convoRepo.NewGormConvoRepo(a.DB, a.RC, time.Duration(a.Config.BrainConfig.MsgTTLMins*int64(time.Minute)))
	a.UserRepo = userRepo.NewGormUserRepo(a.DB)

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
	deps.ConversationService = conversation.New(
		*deps.Configs,
		deps.Mux,
		deps.DeviceRegistry,
		deps.Logger,
		a.ConversationRepo,
	) // Create conversation service

	a.ServerDeps = deps

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
