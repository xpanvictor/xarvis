package user

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"gorm.io/gorm"
)

// OffTimeRangeEntity represents the database entity for OffTimeRange
type OffTimeRangeEntity struct {
	Start time.Time `gorm:"column:start_time"`
	End   time.Time `gorm:"column:end_time"`
	Label string    `gorm:"column:label"`
}

// UserEntity represents the database entity for User with GORM tags
type UserEntity struct {
	ID          string               `gorm:"primaryKey;type:char(36);not null"`
	DisplayName string               `gorm:"column:display_name;type:varchar(255);not null"`
	Email       string               `gorm:"uniqueIndex;type:varchar(191);not null"`
	Timezone    string               `gorm:"type:varchar(64);default:UTC"`
	Settings    json.RawMessage      `gorm:"type:json"`
	OffTimes    []OffTimeRangeEntity `gorm:"type:json;column:off_times"`
	Password    string               `gorm:"column:password_hash;type:char(60);not null"`
	CreatedAt   time.Time            `gorm:"autoCreateTime(3)"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime(3)"`
	DeletedAt   gorm.DeletedAt       `gorm:"index"` // For soft delete
}

// TableName returns the table name for GORM
func (UserEntity) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook to ensure UUID is set
func (u *UserEntity) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// ToDomain converts UserEntity to domain User
func (u *UserEntity) ToDomain() *user.User {
	offTimes := make([]user.OffTimeRange, len(u.OffTimes))
	for i, ot := range u.OffTimes {
		offTimes[i] = user.OffTimeRange{
			Start: ot.Start,
			End:   ot.End,
			Label: ot.Label,
		}
	}

	return &user.User{
		ID:          u.ID,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		Timezone:    u.Timezone,
		Settings:    u.Settings,
		OffTimes:    offTimes,
		Password:    u.Password,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// FromDomain converts domain User to UserEntity
func (u *UserEntity) FromDomain(domainUser *user.User) {
	offTimes := make([]OffTimeRangeEntity, len(domainUser.OffTimes))
	for i, ot := range domainUser.OffTimes {
		offTimes[i] = OffTimeRangeEntity{
			Start: ot.Start,
			End:   ot.End,
			Label: ot.Label,
		}
	}

	u.ID = domainUser.ID
	u.DisplayName = domainUser.DisplayName
	u.Email = domainUser.Email
	u.Timezone = domainUser.Timezone
	u.Settings = domainUser.Settings
	u.OffTimes = offTimes
	u.Password = domainUser.Password
	u.CreatedAt = domainUser.CreatedAt
	u.UpdatedAt = domainUser.UpdatedAt
}

// NewUserEntityFromDomain creates a new UserEntity from domain User
func NewUserEntityFromDomain(domainUser *user.User) *UserEntity {
	entity := &UserEntity{}
	entity.FromDomain(domainUser)
	return entity
}
