package handlers

import (
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/internal/types"
)

// Response wrapper types for Swagger documentation

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Something went wrong"`
	Details string `json:"details,omitempty" example:"Validation error details"`
}

// RegisterResponse represents the response for user registration
type RegisterResponse struct {
	Message string            `json:"message" example:"User registered successfully"`
	User    user.UserResponse `json:"user"`
}

// LoginResponse represents the response for user login
type LoginResponse struct {
	Message string            `json:"message" example:"Login successful"`
	User    user.UserResponse `json:"user"`
	Tokens  user.AuthTokens   `json:"tokens"`
}

// RefreshTokenResponse represents the response for token refresh
type RefreshTokenResponse struct {
	Message string          `json:"message" example:"Token refreshed successfully"`
	Tokens  user.AuthTokens `json:"tokens"`
}

// ProfileResponse represents the response for getting user profile
type ProfileResponse struct {
	User user.UserResponse `json:"user"`
}

// UpdateProfileResponse represents the response for updating user profile
type UpdateProfileResponse struct {
	Message string            `json:"message" example:"Profile updated successfully"`
	User    user.UserResponse `json:"user"`
}

// DeleteAccountResponse represents the response for account deletion
type DeleteAccountResponse struct {
	Message string `json:"message" example:"Account deleted successfully"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Total  int64 `json:"total" example:"150"`
	Offset int   `json:"offset" example:"0"`
	Limit  int   `json:"limit" example:"20"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []user.UserResponse `json:"users"`
	Pagination PaginationInfo      `json:"pagination"`
}

// UserByIDResponse represents the response for getting user by ID
type UserByIDResponse struct {
	User user.UserResponse `json:"user"`
}

// RefreshTokenRequest represents the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"jwt-refresh-token-here"`
}

type MessageResponse struct {
	Message types.Message `json:"message"`
}

// MemoryResponse represents the response for creating a memory
type MemoryResponse struct {
	Memory types.Memory `json:"memory"`
}

// ConversationResponse represents the response for getting conversation
type ConversationResponse struct {
	Conversation types.Conversation `json:"conversation"`
}
