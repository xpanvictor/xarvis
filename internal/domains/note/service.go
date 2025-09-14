package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// Common errors
var (
	ErrNoteNotFound       = errors.New("note not found")
	ErrUnauthorizedAccess = errors.New("unauthorized access to note")
	ErrInvalidNoteData    = errors.New("invalid note data")
)

// NoteService defines the interface for note business logic
type NoteService interface {
	// Note management
	CreateNote(ctx context.Context, userID string, req CreateNoteRequest) (*NoteResponse, error)
	GetNote(ctx context.Context, userID, noteID string) (*NoteResponse, error)
	UpdateNote(ctx context.Context, userID, noteID string, req UpdateNoteRequest) (*NoteResponse, error)
	DeleteNote(ctx context.Context, userID, noteID string) error

	// Note listing and filtering
	ListUserNotes(ctx context.Context, userID string, filters ListNotesRequest) ([]NoteResponse, int64, error)
	ListAllNotes(ctx context.Context, filters ListNotesRequest) ([]NoteResponse, int64, error)

	// Search functionality
	SearchNotes(ctx context.Context, userID, query string, tags []string, offset, limit int) ([]NoteResponse, int64, error)
	GetNotesByTags(ctx context.Context, userID string, tags []string, offset, limit int) ([]NoteResponse, int64, error)

	// Project-related notes (logs)
	GetProjectNotes(ctx context.Context, userID, projectID string, offset, limit int) ([]NoteResponse, int64, error)
	CreateProjectNote(ctx context.Context, userID, projectID string, req CreateNoteRequest) (*NoteResponse, error)
}

type noteService struct {
	repository NoteRepository
	logger     *Logger.Logger
}

// CreateNote implements NoteService
func (s *noteService) CreateNote(ctx context.Context, userID string, req CreateNoteRequest) (*NoteResponse, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Validate content
	if req.Content == "" {
		return nil, ErrInvalidNoteData
	}

	// Create note using domain constructor
	note := NewNote(userUUID, req)

	if err := s.repository.Create(note); err != nil {
		s.logger.Errorf("error creating note: %v", err)
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	s.logger.Infof("note created successfully: %s for user %s", note.ID, userID)
	response := note.ToResponse()
	return &response, nil
}

// GetNote implements NoteService
func (s *noteService) GetNote(ctx context.Context, userID, noteID string) (*NoteResponse, error) {
	note, err := s.repository.GetByID(noteID)
	if err != nil {
		if errors.Is(err, ErrNoteNotFound) {
			return nil, ErrNoteNotFound
		}
		s.logger.Errorf("error getting note: %v", err)
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	// Check if user has access to this note
	if note.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	response := note.ToResponse()
	return &response, nil
}

// UpdateNote implements NoteService
func (s *noteService) UpdateNote(ctx context.Context, userID, noteID string, req UpdateNoteRequest) (*NoteResponse, error) {
	// First check if note exists and user has access
	existing, err := s.repository.GetByID(noteID)
	if err != nil {
		if errors.Is(err, ErrNoteNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	if existing.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Validate update data
	if req.Content != nil && *req.Content == "" {
		return nil, ErrInvalidNoteData
	}

	// Update the note
	updatedNote, err := s.repository.Update(noteID, req)
	if err != nil {
		s.logger.Errorf("error updating note: %v", err)
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	s.logger.Infof("note updated successfully: %s", noteID)
	response := updatedNote.ToResponse()
	return &response, nil
}

// DeleteNote implements NoteService
func (s *noteService) DeleteNote(ctx context.Context, userID, noteID string) error {
	// First check if note exists and user has access
	existing, err := s.repository.GetByID(noteID)
	if err != nil {
		if errors.Is(err, ErrNoteNotFound) {
			return ErrNoteNotFound
		}
		return fmt.Errorf("failed to get note: %w", err)
	}

	if existing.UserID.String() != userID {
		return ErrUnauthorizedAccess
	}

	if err := s.repository.Delete(noteID); err != nil {
		s.logger.Errorf("error deleting note: %v", err)
		return fmt.Errorf("failed to delete note: %w", err)
	}

	s.logger.Infof("note deleted successfully: %s", noteID)
	return nil
}

// ListUserNotes implements NoteService
func (s *noteService) ListUserNotes(ctx context.Context, userID string, filters ListNotesRequest) ([]NoteResponse, int64, error) {
	notes, total, err := s.repository.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Errorf("error listing user notes: %v", err)
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}

	responses := make([]NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = note.ToResponse()
	}

	return responses, total, nil
}

// ListAllNotes implements NoteService (admin only)
func (s *noteService) ListAllNotes(ctx context.Context, filters ListNotesRequest) ([]NoteResponse, int64, error) {
	notes, total, err := s.repository.List(filters)
	if err != nil {
		s.logger.Errorf("error listing all notes: %v", err)
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}

	responses := make([]NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = note.ToResponse()
	}

	return responses, total, nil
}

// SearchNotes implements NoteService
func (s *noteService) SearchNotes(ctx context.Context, userID, query string, tags []string, offset, limit int) ([]NoteResponse, int64, error) {
	notes, total, err := s.repository.Search(userID, query, tags, offset, limit)
	if err != nil {
		s.logger.Errorf("error searching notes: %v", err)
		return nil, 0, fmt.Errorf("failed to search notes: %w", err)
	}

	responses := make([]NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = note.ToResponse()
	}

	return responses, total, nil
}

// GetNotesByTags implements NoteService
func (s *noteService) GetNotesByTags(ctx context.Context, userID string, tags []string, offset, limit int) ([]NoteResponse, int64, error) {
	notes, total, err := s.repository.GetByTags(userID, tags, offset, limit)
	if err != nil {
		s.logger.Errorf("error getting notes by tags: %v", err)
		return nil, 0, fmt.Errorf("failed to get notes by tags: %w", err)
	}

	responses := make([]NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = note.ToResponse()
	}

	return responses, total, nil
}

// GetProjectNotes implements NoteService
func (s *noteService) GetProjectNotes(ctx context.Context, userID, projectID string, offset, limit int) ([]NoteResponse, int64, error) {
	// Note: We may want to add project ownership validation here in the future
	// For now, we trust that the calling handler has validated access to the project

	notes, _, err := s.repository.GetByProjectID(projectID, offset, limit)
	if err != nil {
		s.logger.Errorf("error getting project notes: %v", err)
		return nil, 0, fmt.Errorf("failed to get project notes: %w", err)
	}

	// Filter notes to only return those owned by the requesting user
	var userNotes []Note
	for _, note := range notes {
		if note.UserID.String() == userID {
			userNotes = append(userNotes, note)
		}
	}

	responses := make([]NoteResponse, len(userNotes))
	for i, note := range userNotes {
		responses[i] = note.ToResponse()
	}

	// Return filtered count
	return responses, int64(len(userNotes)), nil
}

// CreateProjectNote implements NoteService
func (s *noteService) CreateProjectNote(ctx context.Context, userID, projectID string, req CreateNoteRequest) (*NoteResponse, error) {
	// Validate content
	if req.Content == "" {
		return nil, ErrInvalidNoteData
	}

	// Create project note using repository method
	note, err := s.repository.CreateProjectNote(projectID, userID, req)
	if err != nil {
		s.logger.Errorf("error creating project note: %v", err)
		return nil, fmt.Errorf("failed to create project note: %w", err)
	}

	s.logger.Infof("project note created successfully: %s for project %s by user %s", note.ID, projectID, userID)
	response := note.ToResponse()
	return &response, nil
}

// NewNoteService creates a new note service
func NewNoteService(repository NoteRepository, logger *Logger.Logger) NoteService {
	return &noteService{
		repository: repository,
		logger:     logger,
	}
}
