package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// Common errors
var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrUnauthorizedAccess = errors.New("unauthorized access to project")
	ErrInvalidProjectData = errors.New("invalid project data")
)

// ProjectService defines the interface for project business logic
type ProjectService interface {
	// Project management
	CreateProject(ctx context.Context, userID string, req CreateProjectRequest) (*ProjectResponse, error)
	GetProject(ctx context.Context, userID, projectID string) (*ProjectResponse, error)
	UpdateProject(ctx context.Context, userID, projectID string, req UpdateProjectRequest) (*ProjectResponse, error)
	DeleteProject(ctx context.Context, userID, projectID string) error

	// Project listing and filtering
	ListUserProjects(ctx context.Context, userID string, filters ListProjectsRequest) ([]ProjectResponse, int64, error)
	ListAllProjects(ctx context.Context, filters ListProjectsRequest) ([]ProjectResponse, int64, error)

	// Progress tracking
	AddProgressEvent(ctx context.Context, userID, projectID string, req AddProgressEventRequest) (*ProjectResponse, error)

	// Project status management
	UpdateProjectStatus(ctx context.Context, userID, projectID string, status ProjectStatus) (*ProjectResponse, error)
}

type projectService struct {
	repository ProjectRepository
	logger     *Logger.Logger
}

// CreateProject implements ProjectService
func (s *projectService) CreateProject(ctx context.Context, userID string, req CreateProjectRequest) (*ProjectResponse, error) {
	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create project using domain constructor
	project := NewProject(userUUID, req)

	if err := s.repository.Create(project); err != nil {
		s.logger.Errorf("error creating project: %v", err)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.Infof("project created successfully: %s for user %s", project.ID, userID)
	response := project.ToResponse()
	return &response, nil
}

// GetProject implements ProjectService
func (s *projectService) GetProject(ctx context.Context, userID, projectID string) (*ProjectResponse, error) {
	project, err := s.repository.GetByID(projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		s.logger.Errorf("error getting project: %v", err)
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if user has access to this project
	if project.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	response := project.ToResponse()
	return &response, nil
}

// UpdateProject implements ProjectService
func (s *projectService) UpdateProject(ctx context.Context, userID, projectID string, req UpdateProjectRequest) (*ProjectResponse, error) {
	// First check if project exists and user has access
	existing, err := s.repository.GetByID(projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if existing.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Update the project
	updatedProject, err := s.repository.Update(projectID, req)
	if err != nil {
		s.logger.Errorf("error updating project: %v", err)
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.Infof("project updated successfully: %s", projectID)
	response := updatedProject.ToResponse()
	return &response, nil
}

// DeleteProject implements ProjectService
func (s *projectService) DeleteProject(ctx context.Context, userID, projectID string) error {
	// First check if project exists and user has access
	existing, err := s.repository.GetByID(projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("failed to get project: %w", err)
	}

	if existing.UserID.String() != userID {
		return ErrUnauthorizedAccess
	}

	if err := s.repository.Delete(projectID); err != nil {
		s.logger.Errorf("error deleting project: %v", err)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.Infof("project deleted successfully: %s", projectID)
	return nil
}

// ListUserProjects implements ProjectService
func (s *projectService) ListUserProjects(ctx context.Context, userID string, filters ListProjectsRequest) ([]ProjectResponse, int64, error) {
	projects, total, err := s.repository.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Errorf("error listing user projects: %v", err)
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	responses := make([]ProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = project.ToResponse()
	}

	return responses, total, nil
}

// ListAllProjects implements ProjectService (admin only)
func (s *projectService) ListAllProjects(ctx context.Context, filters ListProjectsRequest) ([]ProjectResponse, int64, error) {
	projects, total, err := s.repository.List(filters)
	if err != nil {
		s.logger.Errorf("error listing all projects: %v", err)
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	responses := make([]ProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = project.ToResponse()
	}

	return responses, total, nil
}

// AddProgressEvent implements ProjectService
func (s *projectService) AddProgressEvent(ctx context.Context, userID, projectID string, req AddProgressEventRequest) (*ProjectResponse, error) {
	// First check if project exists and user has access
	existing, err := s.repository.GetByID(projectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if existing.UserID.String() != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Add progress event
	updatedProject, err := s.repository.AddProgressEvent(projectID, req)
	if err != nil {
		s.logger.Errorf("error adding progress event: %v", err)
		return nil, fmt.Errorf("failed to add progress event: %w", err)
	}

	s.logger.Infof("progress event added to project %s: %s", projectID, req.Kind)
	response := updatedProject.ToResponse()
	return &response, nil
}

// UpdateProjectStatus implements ProjectService
func (s *projectService) UpdateProjectStatus(ctx context.Context, userID, projectID string, status ProjectStatus) (*ProjectResponse, error) {
	// Create an update request with just the status
	req := UpdateProjectRequest{
		Status: &status,
	}

	_, err := s.UpdateProject(ctx, userID, projectID, req)
	if err != nil {
		return nil, err
	}

	// Add a progress event for the status change
	var eventKind ProgressEventKind
	switch status {
	case StatusInProgress:
		eventKind = EventStarted
	case StatusBlocked:
		eventKind = EventBlocked
	case StatusDone:
		eventKind = EventCompleted
	default:
		eventKind = EventComment
	}

	// Add progress event
	progressReq := AddProgressEventRequest{
		Kind: eventKind,
		By:   "user",
		Memo: fmt.Sprintf("Status changed to %s", status),
	}

	return s.AddProgressEvent(ctx, userID, projectID, progressReq)
}

// NewProjectService creates a new project service
func NewProjectService(repository ProjectRepository, logger *Logger.Logger) ProjectService {
	return &projectService{
		repository: repository,
		logger:     logger,
	}
}
