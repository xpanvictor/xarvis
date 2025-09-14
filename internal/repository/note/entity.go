package note

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/note"
	"gorm.io/gorm"
)

// TagList is a custom type for handling JSON serialization of string slices
type TagList []string

// Value implements driver.Valuer interface for GORM
func (t TagList) Value() (driver.Value, error) {
	if len(t) == 0 {
		return "[]", nil
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner interface for GORM
func (t *TagList) Scan(value interface{}) error {
	if value == nil {
		*t = TagList{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		*t = TagList{}
		return nil
	}
}

// NoteEntity represents the database entity for Note with GORM tags
type NoteEntity struct {
	ID        uuid.UUID  `gorm:"primaryKey;type:char(36);not null"`
	UserID    uuid.UUID  `gorm:"column:user_id;type:char(36);not null;index"`
	ProjectID *uuid.UUID `gorm:"column:project_id;type:char(36);index"` // Optional foreign key to projects
	Content   string     `gorm:"column:content;type:text;not null"`
	Tags      TagList    `gorm:"type:json;column:tags"`
	CreatedAt time.Time  `gorm:"autoCreateTime(3)"`
}

// TableName returns the table name for GORM
func (NoteEntity) TableName() string {
	return "notes"
}

// BeforeCreate is a GORM hook to ensure UUID is set
func (n *NoteEntity) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// ToDomain converts NoteEntity to domain Note
func (n *NoteEntity) ToDomain() *note.Note {
	tags := []string(n.Tags)
	if tags == nil {
		tags = []string{}
	}

	return &note.Note{
		ID:        n.ID,
		UserID:    n.UserID,
		ProjectID: n.ProjectID,
		Content:   n.Content,
		Tags:      tags,
		CreatedAt: n.CreatedAt,
	}
}

// FromDomain converts domain Note to NoteEntity
func (n *NoteEntity) FromDomain(domainNote *note.Note) {
	n.ID = domainNote.ID
	n.UserID = domainNote.UserID
	n.ProjectID = domainNote.ProjectID
	n.Content = domainNote.Content
	n.Tags = TagList(domainNote.Tags)
	n.CreatedAt = domainNote.CreatedAt
}

// NewNoteEntityFromDomain creates a new NoteEntity from domain Note
func NewNoteEntityFromDomain(domainNote *note.Note) *NoteEntity {
	entity := &NoteEntity{}
	entity.FromDomain(domainNote)
	return entity
}
