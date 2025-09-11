package user

import (
	"errors"
	"fmt"

	"github.com/xpanvictor/xarvis/internal/domains/user"
	"gorm.io/gorm"
)

type GormUserRepo struct {
	db *gorm.DB
}

// Create implements user.UserRepository
func (g *GormUserRepo) Create(u *user.User) error {
	entity := NewUserEntityFromDomain(u)
	if err := g.db.Create(entity).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update domain object with any changes from database (like auto-generated fields)
	*u = *entity.ToDomain()
	return nil
}

// GetByID implements user.UserRepository
func (g *GormUserRepo) GetByID(id string) (*user.User, error) {
	var entity UserEntity
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return entity.ToDomain(), nil
}

// GetByEmail implements user.UserRepository
func (g *GormUserRepo) GetByEmail(email string) (*user.User, error) {
	var entity UserEntity
	if err := g.db.Where("email = ?", email).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return entity.ToDomain(), nil
}

// Update implements user.UserRepository
func (g *GormUserRepo) Update(id string, updates user.UpdateUserRequest) (*user.User, error) {
	var entity UserEntity

	// First, get the existing user
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user for update: %w", err)
	}

	// Apply updates only for non-nil fields
	updateMap := make(map[string]interface{})

	if updates.DisplayName != nil {
		updateMap["display_name"] = *updates.DisplayName
	}
	if updates.Timezone != nil {
		updateMap["timezone"] = *updates.Timezone
	}
	if updates.Settings != nil {
		updateMap["settings"] = *updates.Settings
	}
	if updates.OffTimes != nil {
		// Convert domain OffTimes to entity OffTimes
		offTimesEntity := make([]OffTimeRangeEntity, len(*updates.OffTimes))
		for i, ot := range *updates.OffTimes {
			offTimesEntity[i] = OffTimeRangeEntity{
				Start: ot.Start,
				End:   ot.End,
				Label: ot.Label,
			}
		}
		updateMap["off_times"] = offTimesEntity
	}

	// Perform the update
	if len(updateMap) > 0 {
		if err := g.db.Model(&entity).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Return the updated user
	if err := g.db.Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	return entity.ToDomain(), nil
}

// Delete implements user.UserRepository (soft delete)
func (g *GormUserRepo) Delete(id string) error {
	result := g.db.Where("id = ?", id).Delete(&UserEntity{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

// List implements user.UserRepository
func (g *GormUserRepo) List(offset, limit int) ([]user.User, int64, error) {
	var entities []UserEntity
	var total int64

	// Get total count
	if err := g.db.Model(&UserEntity{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	if err := g.db.Offset(offset).Limit(limit).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert entities to domain objects
	users := make([]user.User, len(entities))
	for i, entity := range entities {
		users[i] = *entity.ToDomain()
	}

	return users, total, nil
}

// EmailExists implements user.UserRepository
func (g *GormUserRepo) EmailExists(email string) (bool, error) {
	var count int64
	if err := g.db.Model(&UserEntity{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return count > 0, nil
}

// NewGormUserRepo creates a new GORM-based user repository
func NewGormUserRepo(db *gorm.DB) user.UserRepository {
	return &GormUserRepo{db: db}
}
