package usergroup

import (
	"context"
	"errors"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

type userGroupRepository struct {
	db *gorm.DB
}

var _ ports.UserGroupRepository = (*userGroupRepository)(nil)

func NewUserGroupRepository(db *gorm.DB) ports.UserGroupRepository {
	return &userGroupRepository{db: db}
}

func NewGroupRepository(db *gorm.DB) ports.UserGroupRepository {
	return NewUserGroupRepository(db)
}

func (r *userGroupRepository) AddUser(ctx context.Context, groupID uint, userID uint) error {

	relation := models.UserGroup{
		UserID:  userID,
		GroupID: groupID,
	}

	err := r.db.
		WithContext(ctx).
		Create(&relation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrMembershipExists
		}

		return err
	}

	return nil
}

func (r *userGroupRepository) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	result := r.db.
		WithContext(ctx).
		Where(
			"group_id = ? AND user_id = ?",
			groupID,
			userID,
		).
		Delete(&models.UserGroup{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrMembershipNotFound
	}

	return nil
}

func (r *userGroupRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.
		WithContext(ctx).
		Model(&models.UserGroup{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}

func (r *userGroupRepository) FindAllUserIDsByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	userIDs := make([]uint, 0)
	err := r.db.
		WithContext(ctx).
		Model(&models.UserGroup{}).
		Where("group_id = ?", groupID).
		Order("user_id ASC").
		Pluck("user_id", &userIDs).Error

	if err != nil {
		return nil, err
	}

	return userIDs, nil
}

func (r *userGroupRepository) FindAllAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
	var storedAccessTypes []string

	err := r.db.
		WithContext(ctx).
		Table("group_databases AS gd").
		Joins(
			"JOIN user_groups AS ug ON ug.group_id = gd.group_id",
		).
		Where(
			"ug.user_id = ? AND gd.database_id = ?",
			userID,
			databaseID,
		).
		Distinct("gd.access_type").
		Pluck("gd.access_type", &storedAccessTypes).Error

	if err != nil {
		return nil, err
	}

	accessTypes := make([]domain.AccessType, 0, len(storedAccessTypes))
	for _, storedAccessType := range storedAccessTypes {
		accessTypes = append(
			accessTypes,
			domain.AccessType(storedAccessType),
		)
	}

	return accessTypes, nil
}

func (r *userGroupRepository) FindAllUserIDsByDatabaseID(ctx context.Context, databaseID uint) ([]uint, error) {
	var userIDs []uint

	err := r.db.
		WithContext(ctx).
		Table("user_groups").
		Select("DISTINCT user_groups.user_id").
		Joins("JOIN group_databases ON group_databases.group_id = user_groups.group_id").
		Where("group_databases.database_id = ?", databaseID).
		Pluck("user_groups.user_id", &userIDs).Error

	return userIDs, err
}

func (r *userGroupRepository) HasGroupWithOnlyThisDatabase(ctx context.Context, databaseID uint) (bool, error) {
	var exists bool

	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM group_databases AS target
			WHERE target.database_id = ?
			AND NOT EXISTS (
				SELECT 1
				FROM group_databases AS other
				WHERE other.group_id = target.group_id
				AND other.database_id <> target.database_id
			)
		)
	`, databaseID).Scan(&exists).Error

	return exists, err
}
