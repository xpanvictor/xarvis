package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"golang.org/x/crypto/bcrypt"
)

// Common errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

// AuthTokens represents JWT tokens for authentication
// @Description JWT authentication tokens
type AuthTokens struct {
	AccessToken  string    `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string    `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt    time.Time `json:"expiresAt" example:"2023-01-02T12:00:00Z"`
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// UserService defines the interface for user business logic
type UserService interface {
	// Authentication
	Register(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
	Login(ctx context.Context, req LoginRequest) (*UserResponse, *AuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error)

	// Profile management
	GetProfile(ctx context.Context, userID string) (*UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateUserRequest) (*UserResponse, error)
	DeleteAccount(ctx context.Context, userID string) error

	// Admin operations
	ListUsers(ctx context.Context, offset, limit int) ([]UserResponse, int64, error)
	GetUserByID(ctx context.Context, userID string) (*UserResponse, error)

	// Token validation
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
}

type userService struct {
	repository UserRepository
	logger     *Logger.Logger
	jwtSecret  string
	tokenTTL   time.Duration
}

// Register implements UserService
func (s *userService) Register(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	// Check if email already exists
	exists, err := s.repository.EmailExists(req.Email)
	if err != nil {
		s.logger.Errorf("error checking email existence: %v", err)
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorf("error hashing password: %v", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default timezone if not provided
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Create user
	user := &User{
		ID:          uuid.New().String(),
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Password:    string(hashedPassword),
		Timezone:    timezone,
		Settings:    req.Settings,
		OffTimes:    []OffTimeRange{},
	}

	if err := s.repository.Create(user); err != nil {
		s.logger.Errorf("error creating user: %v", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Infof("user registered successfully: %s (%s)", user.ID, user.Email)
	response := user.ToResponse()
	return &response, nil
}

// Login implements UserService
func (s *userService) Login(ctx context.Context, req LoginRequest) (*UserResponse, *AuthTokens, error) {
	// Get user by email
	user, err := s.repository.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		s.logger.Errorf("error getting user by email: %v", err)
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.generateTokens(user.ID, user.Email)
	if err != nil {
		s.logger.Errorf("error generating tokens: %v", err)
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Infof("user logged in successfully: %s (%s)", user.ID, user.Email)
	response := user.ToResponse()
	return &response, tokens, nil
}

// RefreshToken implements UserService
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Verify user still exists
	user, err := s.repository.GetByID(claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	newTokens, err := s.generateTokens(user.ID, user.Email)
	if err != nil {
		s.logger.Errorf("error generating new tokens: %v", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return newTokens, nil
}

// GetProfile implements UserService
func (s *userService) GetProfile(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.repository.GetByID(userID)
	if err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

// UpdateProfile implements UserService
func (s *userService) UpdateProfile(ctx context.Context, userID string, req UpdateUserRequest) (*UserResponse, error) {
	updatedUser, err := s.repository.Update(userID, req)
	if err != nil {
		s.logger.Errorf("error updating user profile: %v", err)
		return nil, err
	}

	s.logger.Infof("user profile updated: %s", userID)
	response := updatedUser.ToResponse()
	return &response, nil
}

// DeleteAccount implements UserService
func (s *userService) DeleteAccount(ctx context.Context, userID string) error {
	if err := s.repository.Delete(userID); err != nil {
		s.logger.Errorf("error deleting user account: %v", err)
		return err
	}

	s.logger.Infof("user account deleted: %s", userID)
	return nil
}

// ListUsers implements UserService
func (s *userService) ListUsers(ctx context.Context, offset, limit int) ([]UserResponse, int64, error) {
	users, total, err := s.repository.List(offset, limit)
	if err != nil {
		s.logger.Errorf("error listing users: %v", err)
		return nil, 0, err
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	return responses, total, nil
}

// GetUserByID implements UserService
func (s *userService) GetUserByID(ctx context.Context, userID string) (*UserResponse, error) {
	return s.GetProfile(ctx, userID)
}

// ValidateToken implements UserService
func (s *userService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Helper function to generate JWT tokens
func (s *userService) generateTokens(userID, email string) (*AuthTokens, error) {
	expiresAt := time.Now().Add(s.tokenTTL)

	// Create access token
	accessClaims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	// Create refresh token (longer expiry)
	refreshExpiresAt := time.Now().Add(s.tokenTTL * 24) // 24x longer
	refreshClaims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
	}, nil
}

// NewUserService creates a new user service
func NewUserService(repository UserRepository, logger *Logger.Logger, jwtSecret string, tokenTTL time.Duration) UserService {
	if tokenTTL == 0 {
		tokenTTL = 24 * time.Hour // default 24 hours
	}

	return &userService{
		repository: repository,
		logger:     logger,
		jwtSecret:  jwtSecret,
		tokenTTL:   tokenTTL,
	}
}
