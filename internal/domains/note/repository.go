package note

import (
	"time"

	"github.com/google/uuid"
)

// Note represents a note in the system (pure domain model)
// @Description Note entity for storing user notes and snippets
type Note struct {
	ID        uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    uuid.UUID  `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	ProjectID *uuid.UUID `json:"projectId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"` // Optional: for project logs
	Content   string     `json:"content" example:"Important reminder about project deadline"`
	Tags      []string   `json:"tags" example:"reminder,deadline,important"`
	CreatedAt time.Time  `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

// CreateNoteRequest represents the data needed to create a new note
// @Description Request body for note creation
type CreateNoteRequest struct {
	ProjectID *string  `json:"projectId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"` // Optional: for project logs
	Content   string   `json:"content" binding:"required,min=1" example:"Important reminder about project deadline"`
	Tags      []string `json:"tags,omitempty" example:"reminder,deadline"`
}

// UpdateNoteRequest represents the data that can be updated for a note
// @Description Request body for updating note
type UpdateNoteRequest struct {
	ProjectID *string   `json:"projectId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"` // Optional: for project logs
	Content   *string   `json:"content,omitempty" binding:"omitempty,min=1" example:"Updated note content"`
	Tags      *[]string `json:"tags,omitempty" example:"reminder,updated"`
}

// NoteResponse represents a note without sensitive information
// @Description Note information returned in API responses
type NoteResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string    `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	ProjectID *string   `json:"projectId,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Content   string    `json:"content" example:"Important reminder about project deadline"`
	Tags      []string  `json:"tags" example:"reminder,deadline,important"`
	CreatedAt time.Time `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

// ListNotesRequest represents filters for listing notes
// @Description Query parameters for listing notes
type ListNotesRequest struct {
	Tags    []string `form:"tags" example:"reminder,important"`
	Search  string   `form:"search" example:"deadline"`
	Offset  int      `form:"offset" example:"0"`
	Limit   int      `form:"limit" example:"10"`
	OrderBy string   `form:"orderBy" example:"created_at"` // created_at, content
	Order   string   `form:"order" example:"desc"`         // asc, desc
}

// ToResponse converts a Note to NoteResponse
func (n *Note) ToResponse() NoteResponse {
	var projectID *string
	if n.ProjectID != nil {
		projID := n.ProjectID.String()
		projectID = &projID
	}

	return NoteResponse{
		ID:        n.ID.String(),
		UserID:    n.UserID.String(),
		ProjectID: projectID,
		Content:   n.Content,
		Tags:      n.Tags,
		CreatedAt: n.CreatedAt,
	}
}

// NewNote creates a new note with generated ID
func NewNote(userID uuid.UUID, req CreateNoteRequest) *Note {
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	var projectID *uuid.UUID
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}

	return &Note{
		ID:        uuid.New(),
		UserID:    userID,
		ProjectID: projectID,
		Content:   req.Content,
		Tags:      tags,
		CreatedAt: time.Now(),
	}
}

// NoteRepository defines the interface for note data operations
type NoteRepository interface {
	// Create a new note
	Create(note *Note) error

	// Get note by ID
	GetByID(id string) (*Note, error)

	// Get notes by user ID with optional filters
	GetByUserID(userID string, filters ListNotesRequest) ([]Note, int64, error)

	// Update note
	Update(id string, updates UpdateNoteRequest) (*Note, error)

	// Delete note (hard delete for notes)
	Delete(id string) error

	// Search notes by content and tags
	Search(userID string, query string, tags []string, offset, limit int) ([]Note, int64, error)

	// List all notes with pagination and filters
	List(filters ListNotesRequest) ([]Note, int64, error)

	// Get notes by tags
	GetByTags(userID string, tags []string, offset, limit int) ([]Note, int64, error)

	// Project-related note operations
	GetByProjectID(projectID string, offset, limit int) ([]Note, int64, error)
	CreateProjectNote(projectID, userID string, req CreateNoteRequest) (*Note, error)
}
