package user

import (
	"context"
	"errors"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

var _ ports.UserRepository = (*userRepository)(nil)

func NewUserRepository(db *gorm.DB) ports.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) List(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	var storedUsers []models.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.
		WithContext(ctx).
		Select("id", "username", "role").
		Order("id ASC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&storedUsers).Error; err != nil {
		return nil, 0, err
	}

	users := make([]domain.User, 0, len(storedUsers))

	for _, storedUser := range storedUsers {
		users = append(users, toDomain(storedUser))
	}

	return users, total, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (domain.User, error) {
	var storedUser models.User

	err := r.db.
		WithContext(ctx).
		Select("id", "username", "role").
		First(&storedUser, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, err
	}

	return toDomain(storedUser), nil
}

func (r *userRepository) Create(ctx context.Context, user domain.User, groupIDs []uint) (domain.User, error) {
	var created domain.User

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		storedGroups := make([]models.Group, 0, len(groupIDs))

		for _, groupID := range groupIDs {
			storedGroups = append(storedGroups, models.Group{
				ID: groupID,
			})
		}

		storedUser := models.User{
			Username: user.Username,
			Password: user.Password,
			Role:     user.Role,
			Groups:   storedGroups,
		}

		if err := tx.Omit("Groups").Create(&storedUser).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrUsernameTaken
			}

			return err
		}

		relations := make([]models.UserGroup, 0, len(groupIDs))

		for _, groupID := range groupIDs {
			relations = append(relations, models.UserGroup{
				UserID:  storedUser.ID,
				GroupID: groupID,
			})
		}

		if err := tx.Create(&relations).Error; err != nil {
			return err
		}

		created = domain.User{
			ID:       storedUser.ID,
			Username: storedUser.Username,
			Role:     storedUser.Role,
		}

		return nil
	})

	if err != nil {
		return domain.User{}, err
	}

	return created, nil
}

func (r *userRepository) Update(ctx context.Context, id uint, username *string, passwordHash *string) (domain.User, error) {
	updates := make(map[string]any)

	if username != nil {
		updates["username"] = *username
	}

	if passwordHash != nil {
		updates["password"] = *passwordHash
	}

	if len(updates) == 0 {
		return r.FindByID(ctx, id)
	}

	result := r.db.
		WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.User{}, domain.ErrUsernameTaken
		}

		return domain.User{}, result.Error
	}

	if result.RowsAffected == 0 {
		return domain.User{}, domain.ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("user_id = ?", id).
			Delete(&models.UserGroup{}).Error; err != nil {
			return err
		}

		if err := tx.
			Where("user_id = ?", id).
			Delete(&models.Session{}).Error; err != nil {
			return err
		}

		result := tx.Delete(&models.User{}, id)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return domain.ErrUserNotFound
		}

		return nil
	})
}

func (r *userRepository) SetRole(ctx context.Context, id uint, role string) (domain.User, error) {
	result := r.db.
		WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("role", role)

	if result.Error != nil {
		return domain.User{}, result.Error
	}

	if result.RowsAffected == 0 {
		return domain.User{}, domain.ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func toDomain(storedUser models.User) domain.User {
	return domain.User{
		ID:       storedUser.ID,
		Username: storedUser.Username,
		Role:     storedUser.Role,
	}
}

func (r *userRepository) FindByUsernameWithPassword(ctx context.Context, username string) (domain.User, error) {
	var storedUser models.User

	err := r.db.
		WithContext(ctx).
		Where("username = ?", username).
		First(&storedUser).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, err
	}

	return domain.User{
		ID:       storedUser.ID,
		Username: storedUser.Username,
		Password: storedUser.Password,
		Role:     storedUser.Role,
	}, nil
}
