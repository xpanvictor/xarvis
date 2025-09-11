package repository

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
	if err := g.db.Create(u).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID implements user.UserRepository
func (g *GormUserRepo) GetByID(id string) (*user.User, error) {
	var u user.User
	if err := g.db.Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &u, nil
}

// GetByEmail implements user.UserRepository
func (g *GormUserRepo) GetByEmail(email string) (*user.User, error) {
	var u user.User
	if err := g.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

// Update implements user.UserRepository
func (g *GormUserRepo) Update(id string, updates user.UpdateUserRequest) (*user.User, error) {
	var u user.User

	// First, get the existing user
	if err := g.db.Where("id = ?", id).First(&u).Error; err != nil {
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
		updateMap["off_times"] = *updates.OffTimes
	}

	// Perform the update
	if len(updateMap) > 0 {
		if err := g.db.Model(&u).Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Return the updated user
	if err := g.db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	return &u, nil
}

// Delete implements user.UserRepository (soft delete)
func (g *GormUserRepo) Delete(id string) error {
	result := g.db.Where("id = ?", id).Delete(&user.User{})
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
	var users []user.User
	var total int64

	// Get total count
	if err := g.db.Model(&user.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	if err := g.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// EmailExists implements user.UserRepository
func (g *GormUserRepo) EmailExists(email string) (bool, error) {
	var count int64
	if err := g.db.Model(&user.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return count > 0, nil
}

// NewGormUserRepo creates a new GORM-based user repository
func NewGormUserRepo(db *gorm.DB) user.UserRepository {
	return &GormUserRepo{db: db}
}
