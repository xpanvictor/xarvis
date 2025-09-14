package project

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/project"
	"gorm.io/gorm"
)

// ProgressEventEntity represents the database entity for ProgressEvent
type ProgressEventEntity struct {
	At   time.Time `gorm:"column:at"`
	Kind string    `gorm:"column:kind"`
	By   string    `gorm:"column:by"`
	Memo string    `gorm:"column:memo"`
}

// ProgressEventList is a custom type for JSON serialization
type ProgressEventList []ProgressEventEntity

// Value implements driver.Valuer interface for database storage
func (p ProgressEventList) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "[]", nil
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner interface for database retrieval
func (p *ProgressEventList) Scan(value interface{}) error {
	if value == nil {
		*p = ProgressEventList{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan into ProgressEventList")
	}

	return json.Unmarshal(bytes, p)
}

// TagList is a custom type for JSON serialization of string slices
type TagList []string

// Value implements driver.Valuer interface for database storage
func (t TagList) Value() (driver.Value, error) {
	if len(t) == 0 {
		return "[]", nil
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner interface for database retrieval
func (t *TagList) Scan(value interface{}) error {
	if value == nil {
		*t = TagList{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan into TagList")
	}

	return json.Unmarshal(bytes, t)
}

// ProjectEntity represents the database entity for Project with GORM tags
type ProjectEntity struct {
	ID          uuid.UUID         `gorm:"primaryKey;type:char(36);not null"`
	UserID      uuid.UUID         `gorm:"column:user_id;type:char(36);not null;index"`
	Name        string            `gorm:"column:name;type:varchar(255);not null"`
	Description string            `gorm:"column:description;type:text"`
	Status      string            `gorm:"column:status;type:varchar(20);not null;default:'planned'"`
	Priority    string            `gorm:"column:priority;type:varchar(10);not null;default:'med'"`
	Tags        TagList           `gorm:"type:json;column:tags"`
	DueAt       *time.Time        `gorm:"column:due_at"`
	Progress    ProgressEventList `gorm:"type:json;column:progress"`
	CreatedAt   time.Time         `gorm:"autoCreateTime(3)"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime(3)"`
	DeletedAt   gorm.DeletedAt    `gorm:"index"` // For soft delete
}

// TableName returns the table name for GORM
func (ProjectEntity) TableName() string {
	return "projects"
}

// BeforeCreate is a GORM hook to ensure UUID is set
func (p *ProjectEntity) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ToDomain converts ProjectEntity to domain Project
func (p *ProjectEntity) ToDomain() *project.Project {
	tags := []string(p.Tags)
	if tags == nil {
		tags = []string{}
	}

	progressEvents := make([]project.ProgressEvent, len(p.Progress))
	for i, pe := range p.Progress {
		progressEvents[i] = project.ProgressEvent{
			At:   pe.At,
			Kind: project.ProgressEventKind(pe.Kind),
			By:   pe.By,
			Memo: pe.Memo,
		}
	}

	return &project.Project{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Description: p.Description,
		Status:      project.ProjectStatus(p.Status),
		Priority:    project.ProjectPriority(p.Priority),
		Tags:        tags,
		DueAt:       p.DueAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Progress:    progressEvents,
	}
}

// FromDomain converts domain Project to ProjectEntity
func (p *ProjectEntity) FromDomain(domainProject *project.Project) {
	progressEntities := make(ProgressEventList, len(domainProject.Progress))
	for i, pe := range domainProject.Progress {
		progressEntities[i] = ProgressEventEntity{
			At:   pe.At,
			Kind: string(pe.Kind),
			By:   pe.By,
			Memo: pe.Memo,
		}
	}

	p.ID = domainProject.ID
	p.UserID = domainProject.UserID
	p.Name = domainProject.Name
	p.Description = domainProject.Description
	p.Status = string(domainProject.Status)
	p.Priority = string(domainProject.Priority)
	p.Tags = TagList(domainProject.Tags)
	p.DueAt = domainProject.DueAt
	p.Progress = progressEntities
	p.CreatedAt = domainProject.CreatedAt
	p.UpdatedAt = domainProject.UpdatedAt
}

// NewProjectEntityFromDomain creates a new ProjectEntity from domain Project
func NewProjectEntityFromDomain(domainProject *project.Project) *ProjectEntity {
	entity := &ProjectEntity{}
	entity.FromDomain(domainProject)
	return entity
}
