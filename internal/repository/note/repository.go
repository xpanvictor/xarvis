package note

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/note"
	"gorm.io/gorm"
)

type GormNoteRepo struct {
	db *gorm.DB
}

var (
	ErrNoteNotFound    = errors.New("note not found")
	ErrInvalidNoteData = errors.New("invalid note data")
)

// Create implements note.NoteRepository
func (g *GormNoteRepo) Create(n *note.Note) error {
	entity := NewNoteEntityFromDomain(n)
	if err := g.db.Create(entity).Error; err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	// Update domain object with any changes from database (like auto-generated fields)
	*n = *entity.ToDomain()
	return nil
}

// GetByID implements note.NoteRepository
func (g *GormNoteRepo) GetByID(id string) (*note.Note, error) {
	var entity NoteEntity
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, fmt.Errorf("failed to get note by ID: %w", err)
	}
	return entity.ToDomain(), nil
}

// GetByUserID implements note.NoteRepository
func (g *GormNoteRepo) GetByUserID(userID string, filters note.ListNotesRequest) ([]note.Note, int64, error) {
	var entities []NoteEntity
	var total int64

	query := g.db.Model(&NoteEntity{}).Where("user_id = ?", userID)

	// Apply search filter
	if filters.Search != "" {
		query = query.Where("content LIKE ?", "%"+filters.Search+"%")
	}

	// Apply tags filter
	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notes: %w", err)
	}

	// Apply ordering
	orderBy := "created_at"
	if filters.OrderBy != "" && (filters.OrderBy == "created_at" || filters.OrderBy == "content") {
		orderBy = filters.OrderBy
	}

	order := "DESC"
	if filters.Order != "" && (filters.Order == "asc" || filters.Order == "desc") {
		order = filters.Order
	}

	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}

	// Convert entities to domain objects
	notes := make([]note.Note, len(entities))
	for i, entity := range entities {
		notes[i] = *entity.ToDomain()
	}

	return notes, total, nil
}

// Update implements note.NoteRepository
func (g *GormNoteRepo) Update(id string, updates note.UpdateNoteRequest) (*note.Note, error) {
	var entity NoteEntity

	// First, get the existing note
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, fmt.Errorf("failed to get note for update: %w", err)
	}

	// Validate update data
	if updates.Content != nil && *updates.Content == "" {
		return nil, ErrInvalidNoteData
	}

	// Apply updates only for non-nil fields
	updateMap := make(map[string]interface{})

	if updates.Content != nil {
		updateMap["content"] = *updates.Content
	}
	if updates.Tags != nil {
		updateMap["tags"] = *updates.Tags
	}
	if updates.ProjectID != nil {
		if *updates.ProjectID == "" {
			updateMap["project_id"] = nil // Remove project association
		} else {
			if projectUUID, err := uuid.Parse(*updates.ProjectID); err == nil {
				updateMap["project_id"] = projectUUID
			} else {
				return nil, fmt.Errorf("invalid project ID: %w", err)
			}
		}
	} // Perform the update
	if len(updateMap) > 0 {
		if err := g.db.Model(&entity).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update note: %w", err)
		}
	}

	// Return the updated note
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated note: %w", err)
	}

	return entity.ToDomain(), nil
}

// Delete implements note.NoteRepository (hard delete)
func (g *GormNoteRepo) Delete(id string) error {
	result := g.db.Unscoped().Where("id = ?", id).Delete(&NoteEntity{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete note: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNoteNotFound
	}
	return nil
}

// Search implements note.NoteRepository
func (g *GormNoteRepo) Search(userID string, query string, tags []string, offset, limit int) ([]note.Note, int64, error) {
	filters := note.ListNotesRequest{
		Search: query,
		Tags:   tags,
		Offset: offset,
		Limit:  limit,
	}
	return g.GetByUserID(userID, filters)
}

// List implements note.NoteRepository
func (g *GormNoteRepo) List(filters note.ListNotesRequest) ([]note.Note, int64, error) {
	var entities []NoteEntity
	var total int64

	query := g.db.Model(&NoteEntity{})

	// Apply search filter
	if filters.Search != "" {
		query = query.Where("content LIKE ?", "%"+filters.Search+"%")
	}

	// Apply tags filter
	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notes: %w", err)
	}

	// Apply ordering
	orderBy := "created_at"
	if filters.OrderBy != "" && (filters.OrderBy == "created_at" || filters.OrderBy == "content") {
		orderBy = filters.OrderBy
	}

	order := "DESC"
	if filters.Order != "" && (filters.Order == "asc" || filters.Order == "desc") {
		order = filters.Order
	}

	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}

	// Convert entities to domain objects
	notes := make([]note.Note, len(entities))
	for i, entity := range entities {
		notes[i] = *entity.ToDomain()
	}

	return notes, total, nil
}

// GetByTags implements note.NoteRepository
func (g *GormNoteRepo) GetByTags(userID string, tags []string, offset, limit int) ([]note.Note, int64, error) {
	filters := note.ListNotesRequest{
		Tags:   tags,
		Offset: offset,
		Limit:  limit,
	}
	return g.GetByUserID(userID, filters)
}

// GetByProjectID implements note.NoteRepository
func (g *GormNoteRepo) GetByProjectID(projectID string, offset, limit int) ([]note.Note, int64, error) {
	var entities []NoteEntity
	var total int64

	// Parse project UUID
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid project ID: %w", err)
	}

	query := g.db.Model(&NoteEntity{}).Where("project_id = ?", projectUUID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count project notes: %w", err)
	}

	// Apply ordering (newest first for logs)
	query = query.Order("created_at DESC")

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list project notes: %w", err)
	}

	// Convert entities to domain objects
	notes := make([]note.Note, len(entities))
	for i, entity := range entities {
		notes[i] = *entity.ToDomain()
	}

	return notes, total, nil
}

// CreateProjectNote implements note.NoteRepository
func (g *GormNoteRepo) CreateProjectNote(projectID, userID string, req note.CreateNoteRequest) (*note.Note, error) {
	// Parse project UUID
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Parse user UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create note with project association
	n := &note.Note{
		UserID:    userUUID,
		Content:   req.Content,
		Tags:      req.Tags,
		ProjectID: &projectUUID,
	}

	// Create the note
	if err := g.Create(n); err != nil {
		return nil, fmt.Errorf("failed to create project note: %w", err)
	}

	return n, nil
}

// NewGormNoteRepo creates a new GORM-based note repository
func NewGormNoteRepo(db *gorm.DB) note.NoteRepository {
	return &GormNoteRepo{db: db}
}
