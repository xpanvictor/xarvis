package user

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OffTimeRange represents a time range when the user is not available
// @Description Time range when user is not available
type OffTimeRange struct {
	Start time.Time `json:"start" example:"2023-12-25T09:00:00Z"`
	End   time.Time `json:"end" example:"2023-12-25T17:00:00Z"`
	Label string    `json:"label,omitempty" example:"Christmas Day"`
}

// User represents a user in the system (pure domain model)
// @Description User account information
type User struct {
	ID          string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DisplayName string          `json:"displayName" example:"John Doe"`
	Email       string          `json:"email" example:"john@example.com"`
	Timezone    string          `json:"timezone" example:"America/New_York"`
	Settings    json.RawMessage `json:"settings" swaggertype:"object"`
	OffTimes    []OffTimeRange  `json:"offTimes"`
	Password    string          `json:"-"` // Never expose in JSON
	CreatedAt   time.Time       `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time       `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// CreateUserRequest represents the data needed to create a new user
// @Description Request body for user registration
type CreateUserRequest struct {
	DisplayName string          `json:"displayName" binding:"required,min=2,max=100" example:"John Doe"`
	Email       string          `json:"email" binding:"required,email" example:"john@example.com"`
	Password    string          `json:"password" binding:"required,min=8" example:"securePassword123"`
	Timezone    string          `json:"timezone,omitempty" example:"America/New_York"`
	Settings    json.RawMessage `json:"settings,omitempty" swaggertype:"object"`
}

// UpdateUserRequest represents the data that can be updated for a user
// @Description Request body for updating user profile
type UpdateUserRequest struct {
	DisplayName *string          `json:"displayName,omitempty" binding:"omitempty,min=2,max=100" example:"John Smith"`
	Timezone    *string          `json:"timezone,omitempty" example:"Europe/London"`
	Settings    *json.RawMessage `json:"settings,omitempty" swaggertype:"object"`
	OffTimes    *[]OffTimeRange  `json:"offTimes,omitempty"`
}

// LoginRequest represents login credentials
// @Description Request body for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required" example:"securePassword123"`
}

// UserResponse represents a user without sensitive information
// @Description User information returned in API responses (no sensitive data)
type UserResponse struct {
	ID          string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DisplayName string          `json:"displayName" example:"John Doe"`
	Email       string          `json:"email" example:"john@example.com"`
	Timezone    string          `json:"timezone" example:"America/New_York"`
	Settings    json.RawMessage `json:"settings" swaggertype:"object"`
	OffTimes    []OffTimeRange  `json:"offTimes"`
	CreatedAt   time.Time       `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time       `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// ToResponse converts a User to UserResponse (removes sensitive data)
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		Timezone:    u.Timezone,
		Settings:    u.Settings,
		OffTimes:    u.OffTimes,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// NewUser creates a new user with generated ID
func NewUser(req CreateUserRequest, hashedPassword string) *User {
	return &User{
		ID:          uuid.New().String(),
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Password:    hashedPassword,
		Timezone:    req.Timezone,
		Settings:    req.Settings,
		OffTimes:    []OffTimeRange{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create a new user
	Create(user *User) error

	// Get user by ID
	GetByID(id string) (*User, error)

	// Get user by email
	GetByEmail(email string) (*User, error)

	// Update user
	Update(id string, updates UpdateUserRequest) (*User, error)

	// Delete user (soft delete)
	Delete(id string) error

	// List users with pagination
	List(offset, limit int) ([]User, int64, error)

	// Check if email exists
	EmailExists(email string) (bool, error)
}
