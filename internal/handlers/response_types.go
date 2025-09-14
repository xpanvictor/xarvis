package handlers

import (
	"github.com/xpanvictor/xarvis/internal/domains/note"
	"github.com/xpanvictor/xarvis/internal/domains/project"
	"github.com/xpanvictor/xarvis/internal/domains/task"
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

// Project-related responses

// CreateProjectResponse represents the response for project creation
type CreateProjectResponse struct {
	Message string                  `json:"message" example:"Project created successfully"`
	Project project.ProjectResponse `json:"project"`
}

// ProjectResponse represents the response for getting a single project
type ProjectResponse struct {
	Project project.ProjectResponse `json:"project"`
}

// UpdateProjectResponse represents the response for updating a project
type UpdateProjectResponse struct {
	Message string                  `json:"message" example:"Project updated successfully"`
	Project project.ProjectResponse `json:"project"`
}

// ListProjectsResponse represents the response for listing projects
type ListProjectsResponse struct {
	Projects   []project.ProjectResponse `json:"projects"`
	Pagination PaginationInfo            `json:"pagination"`
}

// UpdateProjectStatusRequest represents the request for updating project status
type UpdateProjectStatusRequest struct {
	Status project.ProjectStatus `json:"status" binding:"required" example:"in_progress"`
}

// Note-related responses

// CreateNoteResponse represents the response for note creation
type CreateNoteResponse struct {
	Message string            `json:"message" example:"Note created successfully"`
	Note    note.NoteResponse `json:"note"`
}

// NoteResponse represents the response for getting a single note
type NoteResponse struct {
	Note note.NoteResponse `json:"note"`
}

// UpdateNoteResponse represents the response for updating a note
type UpdateNoteResponse struct {
	Message string            `json:"message" example:"Note updated successfully"`
	Note    note.NoteResponse `json:"note"`
}

// ListNotesResponse represents the response for listing notes
type ListNotesResponse struct {
	Notes      []note.NoteResponse `json:"notes"`
	Pagination PaginationInfo      `json:"pagination"`
}

// SearchNotesResponse represents the response for searching notes
type SearchNotesResponse struct {
	Notes      []note.NoteResponse `json:"notes"`
	Pagination PaginationInfo      `json:"pagination"`
	Query      string              `json:"query,omitempty"`
	Tags       []string            `json:"tags,omitempty"`
}

// Task-related responses

// CreateTaskResponse represents the response for task creation
type CreateTaskResponse struct {
	Message string            `json:"message" example:"Task created successfully"`
	Task    task.TaskResponse `json:"task"`
}

// TaskResponse represents the response for getting a single task
type TaskResponse struct {
	Task task.TaskResponse `json:"task"`
}

// UpdateTaskResponse represents the response for updating a task
type UpdateTaskResponse struct {
	Message string            `json:"message" example:"Task updated successfully"`
	Task    task.TaskResponse `json:"task"`
}

// ListTasksResponse represents the response for listing tasks
type ListTasksResponse struct {
	Tasks      []task.TaskResponse `json:"tasks"`
	Pagination PaginationInfo      `json:"pagination"`
}

// SearchTasksResponse represents the response for searching tasks
type SearchTasksResponse struct {
	Tasks      []task.TaskResponse `json:"tasks"`
	Pagination PaginationInfo      `json:"pagination"`
	Query      string              `json:"query,omitempty"`
}

// BulkUpdateStatusRequest represents the request for bulk status updates
type BulkUpdateStatusRequest struct {
	TaskIDs []string        `json:"taskIds" binding:"required"`
	Status  task.TaskStatus `json:"status" binding:"required"`
}
