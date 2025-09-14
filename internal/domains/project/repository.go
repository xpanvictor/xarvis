package project

import (
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	StatusPlanned    ProjectStatus = "planned"
	StatusInProgress ProjectStatus = "in_progress"
	StatusBlocked    ProjectStatus = "blocked"
	StatusDone       ProjectStatus = "done"
	StatusArchived   ProjectStatus = "archived"
)

// ProjectPriority represents the priority level of a project
type ProjectPriority string

const (
	PriorityLow    ProjectPriority = "low"
	PriorityMed    ProjectPriority = "med"
	PriorityHigh   ProjectPriority = "high"
	PriorityUrgent ProjectPriority = "urgent"
)

// ProgressEventKind represents the type of progress event
type ProgressEventKind string

const (
	EventPlanned   ProgressEventKind = "planned"
	EventStarted   ProgressEventKind = "started"
	EventBlocked   ProgressEventKind = "blocked"
	EventUnblocked ProgressEventKind = "unblocked"
	EventCompleted ProgressEventKind = "completed"
	EventComment   ProgressEventKind = "comment"
)

// ProgressEvent represents a progress update or event in a project
// @Description Progress event tracking project changes
type ProgressEvent struct {
	At   time.Time         `json:"at" example:"2023-01-01T12:00:00Z"`
	Kind ProgressEventKind `json:"kind" example:"started"`
	By   string            `json:"by" example:"user"` // system/assistant/user
	Memo string            `json:"memo" example:"Started working on initial setup"`
}

// Project represents a project in the system (pure domain model)
// @Description Project management entity
type Project struct {
	ID          uuid.UUID       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      uuid.UUID       `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name        string          `json:"name" example:"Website Redesign"`
	Description string          `json:"description" example:"Complete redesign of company website"`
	Status      ProjectStatus   `json:"status" example:"in_progress"`
	Priority    ProjectPriority `json:"priority" example:"high"`
	Tags        []string        `json:"tags" example:"web,design,frontend"`
	DueAt       *time.Time      `json:"dueAt,omitempty" example:"2023-12-31T23:59:59Z"`
	CreatedAt   time.Time       `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time       `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
	Progress    []ProgressEvent `json:"progress"`
}

// CreateProjectRequest represents the data needed to create a new project
// @Description Request body for project creation
type CreateProjectRequest struct {
	Name        string          `json:"name" binding:"required,min=1,max=255" example:"Website Redesign"`
	Description string          `json:"description,omitempty" example:"Complete redesign of company website"`
	Status      ProjectStatus   `json:"status,omitempty" example:"planned"`
	Priority    ProjectPriority `json:"priority,omitempty" example:"med"`
	Tags        []string        `json:"tags,omitempty" example:"web,design"`
	DueAt       *time.Time      `json:"dueAt,omitempty" example:"2023-12-31T23:59:59Z"`
}

// UpdateProjectRequest represents the data that can be updated for a project
// @Description Request body for updating project
type UpdateProjectRequest struct {
	Name        *string          `json:"name,omitempty" binding:"omitempty,min=1,max=255" example:"Website Redesign v2"`
	Description *string          `json:"description,omitempty" example:"Updated description"`
	Status      *ProjectStatus   `json:"status,omitempty" example:"in_progress"`
	Priority    *ProjectPriority `json:"priority,omitempty" example:"high"`
	Tags        *[]string        `json:"tags,omitempty" example:"web,design,urgent"`
	DueAt       *time.Time       `json:"dueAt,omitempty" example:"2023-12-31T23:59:59Z"`
}

// AddProgressEventRequest represents adding a progress event to a project
// @Description Request body for adding progress event
type AddProgressEventRequest struct {
	Kind ProgressEventKind `json:"kind" binding:"required" example:"started"`
	By   string            `json:"by" binding:"required" example:"user"`
	Memo string            `json:"memo,omitempty" example:"Started working on wireframes"`
}

// ProjectResponse represents a project without sensitive information
// @Description Project information returned in API responses
type ProjectResponse struct {
	ID          string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID      string          `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name        string          `json:"name" example:"Website Redesign"`
	Description string          `json:"description" example:"Complete redesign of company website"`
	Status      ProjectStatus   `json:"status" example:"in_progress"`
	Priority    ProjectPriority `json:"priority" example:"high"`
	Tags        []string        `json:"tags" example:"web,design,frontend"`
	DueAt       *time.Time      `json:"dueAt,omitempty" example:"2023-12-31T23:59:59Z"`
	CreatedAt   time.Time       `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time       `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
	Progress    []ProgressEvent `json:"progress"`
}

// ListProjectsRequest represents filters for listing projects
// @Description Query parameters for listing projects
type ListProjectsRequest struct {
	Status   *ProjectStatus   `form:"status" example:"in_progress"`
	Priority *ProjectPriority `form:"priority" example:"high"`
	Tags     []string         `form:"tags" example:"web,design"`
	Offset   int              `form:"offset" example:"0"`
	Limit    int              `form:"limit" example:"10"`
}

// ToResponse converts a Project to ProjectResponse
func (p *Project) ToResponse() ProjectResponse {
	return ProjectResponse{
		ID:          p.ID.String(),
		UserID:      p.UserID.String(),
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		Priority:    p.Priority,
		Tags:        p.Tags,
		DueAt:       p.DueAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Progress:    p.Progress,
	}
}

// NewProject creates a new project with generated ID
func NewProject(userID uuid.UUID, req CreateProjectRequest) *Project {
	now := time.Now()
	status := req.Status
	if status == "" {
		status = StatusPlanned
	}
	priority := req.Priority
	if priority == "" {
		priority = PriorityMed
	}

	project := &Project{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		Tags:        req.Tags,
		DueAt:       req.DueAt,
		CreatedAt:   now,
		UpdatedAt:   now,
		Progress:    []ProgressEvent{},
	}

	// Add initial progress event
	project.Progress = append(project.Progress, ProgressEvent{
		At:   now,
		Kind: EventPlanned,
		By:   "system",
		Memo: "Project created",
	})

	return project
}

// AddProgressEvent adds a new progress event to the project
func (p *Project) AddProgressEvent(event AddProgressEventRequest) {
	p.Progress = append(p.Progress, ProgressEvent{
		At:   time.Now(),
		Kind: event.Kind,
		By:   event.By,
		Memo: event.Memo,
	})
	p.UpdatedAt = time.Now()
}

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	// Create a new project
	Create(project *Project) error

	// Get project by ID
	GetByID(id string) (*Project, error)

	// Get projects by user ID with optional filters
	GetByUserID(userID string, filters ListProjectsRequest) ([]Project, int64, error)

	// Update project
	Update(id string, updates UpdateProjectRequest) (*Project, error)

	// Delete project (soft delete)
	Delete(id string) error

	// Add progress event to project
	AddProgressEvent(projectID string, event AddProgressEventRequest) (*Project, error)

	// List all projects with pagination and filters
	List(filters ListProjectsRequest) ([]Project, int64, error)
}
