package repositories

import (
	"context"
	"errors"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userGroupRepository struct {
	db *gorm.DB
}

var _ ports.UserGroupRepository = (*userGroupRepository)(nil)

func NewUserGroupRepository(db *gorm.DB) ports.UserGroupRepository {
	return &userGroupRepository{db: db}
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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.UserGroup
		err := tx.Where("group_id = ? AND user_id = ?", groupID, userID).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrMembershipNotFound
			}
			return err
		}

		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return err
		}

		if user.Role == domain.RoleUser {
			var count int64
			if err := tx.Model(&models.UserGroup{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
				return err
			}
			if count <= 1 {
				return domain.ErrUserMustHaveGroup
			}
		}

		return tx.Delete(&existing).Error
	})
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

func (r *userGroupRepository) HasUserWithOnlyThisGroup(ctx context.Context, groupID uint) (bool, error) {
	var exists bool

	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM user_groups AS ug
			JOIN users AS u ON u.id = ug.user_id
			WHERE ug.group_id = ?
			  AND u.role = 'user'
			  AND NOT EXISTS (
				  SELECT 1
				  FROM user_groups AS other
				  WHERE other.user_id = ug.user_id
				    AND other.group_id <> ug.group_id
			  )
		)
	`, groupID).Scan(&exists).Error

	return exists, err
}
