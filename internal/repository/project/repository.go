package project

import (
	"errors"
	"fmt"

	"github.com/xpanvictor/xarvis/internal/domains/project"
	"gorm.io/gorm"
)

type GormProjectRepo struct {
	db *gorm.DB
}

var (
	ErrProjectNotFound = errors.New("project not found")
)

// Create implements project.ProjectRepository
func (g *GormProjectRepo) Create(p *project.Project) error {
	entity := NewProjectEntityFromDomain(p)
	if err := g.db.Create(entity).Error; err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Update domain object with any changes from database (like auto-generated fields)
	*p = *entity.ToDomain()
	return nil
}

// GetByID implements project.ProjectRepository
func (g *GormProjectRepo) GetByID(id string) (*project.Project, error) {
	var entity ProjectEntity
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project by ID: %w", err)
	}
	return entity.ToDomain(), nil
}

// GetByUserID implements project.ProjectRepository
func (g *GormProjectRepo) GetByUserID(userID string, filters project.ListProjectsRequest) ([]project.Project, int64, error) {
	var entities []ProjectEntity
	var total int64

	query := g.db.Model(&ProjectEntity{}).Where("user_id = ?", userID)

	// Apply filters
	if filters.Status != nil {
		query = query.Where("status = ?", string(*filters.Status))
	}
	if filters.Priority != nil {
		query = query.Where("priority = ?", string(*filters.Priority))
	}
	if len(filters.Tags) > 0 {
		// For JSON array contains check - this might need database-specific syntax
		for _, tag := range filters.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("created_at DESC").Offset(filters.Offset).Limit(filters.Limit)

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	// Convert entities to domain objects
	projects := make([]project.Project, len(entities))
	for i, entity := range entities {
		projects[i] = *entity.ToDomain()
	}

	return projects, total, nil
}

// Update implements project.ProjectRepository
func (g *GormProjectRepo) Update(id string, updates project.UpdateProjectRequest) (*project.Project, error) {
	var entity ProjectEntity

	// First, get the existing project
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project for update: %w", err)
	}

	// Apply updates only for non-nil fields
	updateMap := make(map[string]interface{})

	if updates.Name != nil {
		updateMap["name"] = *updates.Name
	}
	if updates.Description != nil {
		updateMap["description"] = *updates.Description
	}
	if updates.Status != nil {
		updateMap["status"] = string(*updates.Status)
	}
	if updates.Priority != nil {
		updateMap["priority"] = string(*updates.Priority)
	}
	if updates.Tags != nil {
		updateMap["tags"] = *updates.Tags
	}
	if updates.DueAt != nil {
		updateMap["due_at"] = *updates.DueAt
	}

	// Perform the update
	if len(updateMap) > 0 {
		if err := g.db.Model(&entity).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update project: %w", err)
		}
	}

	// Return the updated project
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated project: %w", err)
	}

	return entity.ToDomain(), nil
}

// Delete implements project.ProjectRepository (soft delete)
func (g *GormProjectRepo) Delete(id string) error {
	result := g.db.Where("id = ?", id).Delete(&ProjectEntity{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// AddProgressEvent implements project.ProjectRepository
func (g *GormProjectRepo) AddProgressEvent(projectID string, event project.AddProgressEventRequest) (*project.Project, error) {
	var entity ProjectEntity

	// First, get the existing project
	if err := g.db.Where("id = ?", projectID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project for progress update: %w", err)
	}

	// Convert to domain, add event, convert back
	domainProject := entity.ToDomain()
	domainProject.AddProgressEvent(event)

	// Update the entity with new progress
	entity.FromDomain(domainProject)

	// Save the updated project
	if err := g.db.Save(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to update project with progress event: %w", err)
	}

	return entity.ToDomain(), nil
}

// List implements project.ProjectRepository
func (g *GormProjectRepo) List(filters project.ListProjectsRequest) ([]project.Project, int64, error) {
	var entities []ProjectEntity
	var total int64

	query := g.db.Model(&ProjectEntity{})

	// Apply filters
	if filters.Status != nil {
		query = query.Where("status = ?", string(*filters.Status))
	}
	if filters.Priority != nil {
		query = query.Where("priority = ?", string(*filters.Priority))
	}
	if len(filters.Tags) > 0 {
		// For JSON array contains check
		for _, tag := range filters.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("created_at DESC").Offset(filters.Offset).Limit(filters.Limit)

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	// Convert entities to domain objects
	projects := make([]project.Project, len(entities))
	for i, entity := range entities {
		projects[i] = *entity.ToDomain()
	}

	return projects, total, nil
}

// NewGormProjectRepo creates a new GORM-based project repository
func NewGormProjectRepo(db *gorm.DB) project.ProjectRepository {
	return &GormProjectRepo{db: db}
}
