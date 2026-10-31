package group

import (
	"context"
	"errors"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
)

type groupRepository struct {
	db *gorm.DB
}

var _ ports.GroupRepository = (*groupRepository)(nil)

func NewGroupRepository(db *gorm.DB) ports.GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	var storedGroups []models.Group
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Group{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Order("id ASC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&storedGroups).Error
	if err != nil {
		return nil, 0, err
	}

	groups := make([]domain.Group, 0, len(storedGroups))
	if len(storedGroups) == 0 {
		return groups, total, nil
	}

	groupIDs := make([]uint, 0, len(storedGroups))
	for _, storedGroup := range storedGroups {
		groupIDs = append(groupIDs, storedGroup.ID)
	}

	var relations []models.GroupDatabase

	err = r.db.WithContext(ctx).Where("group_id IN ?", groupIDs).Order("group_id ASC, database_id ASC").Find(&relations).Error
	if err != nil {
		return nil, 0, err
	}

	accessesByGroup := make(map[uint][]domain.DatabaseAccess)
	for _, relation := range relations {
		accessesByGroup[relation.GroupID] = append(
			accessesByGroup[relation.GroupID],
			domain.DatabaseAccess{
				DatabaseID: relation.DatabaseID,
				AccessType: domain.AccessType(relation.AccessType),
			})
	}

	for _, storedGroup := range storedGroups {
		groups = append(groups, domain.Group{
			ID:               storedGroup.ID,
			Name:             storedGroup.Name,
			DatabaseAccesses: accessesByGroup[storedGroup.ID],
		})
	}

	return groups, total, nil
}

func (r *groupRepository) FindByID(ctx context.Context, id uint) (domain.Group, error) {
	var storedGroup models.Group

	err := r.db.WithContext(ctx).First(&storedGroup, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Group{}, domain.ErrGroupNotFound
		}
		return domain.Group{}, err
	}

	var relations []models.GroupDatabase
	err = r.db.WithContext(ctx).Where("group_id = ?", id).Order(("database_id ASC")).Find(&relations).Error
	if err != nil {
		return domain.Group{}, err
	}

	accesses := make([]domain.DatabaseAccess, 0, len(relations))
	for _, relation := range relations {
		accesses = append(accesses, domain.DatabaseAccess{
			DatabaseID: relation.DatabaseID,
			AccessType: domain.AccessType(relation.AccessType),
		})
	}

	return domain.Group{
		ID:               storedGroup.ID,
		Name:             storedGroup.Name,
		DatabaseAccesses: accesses,
	}, nil
}

func (r *groupRepository) AllExist(ctx context.Context, ids []uint) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}

	uniqueIDs := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		uniqueIDs[id] = struct{}{}
	}

	cleanIDs := make([]uint, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		cleanIDs = append(cleanIDs, id)
	}

	var count int64
	err := r.db.
		WithContext(ctx).
		Model(&models.Group{}).
		Where("id IN ?", cleanIDs).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count == int64(len(cleanIDs)), nil
}

func (r *groupRepository) Save(ctx context.Context, group domain.Group) (domain.Group, error) {
	var created domain.Group
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		storedGroup := models.Group{
			Name: group.Name,
		}

		if err := tx.Create(&storedGroup).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrGroupNameTaken
			}
			return err
		}
		if err := replaceDatabaseAccesses(tx, storedGroup.ID, group.DatabaseAccesses); err != nil {
			return err
		}

		group.ID = storedGroup.ID
		created = group

		return nil
	})

	if err != nil {
		return domain.Group{}, err
	}

	return created, nil
}

func (r *groupRepository) Update(ctx context.Context, group domain.Group) (domain.Group, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Model(&models.Group{}).
			Where("id = ?", group.ID).
			Update("name", group.Name)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return domain.ErrGroupNameTaken
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrGroupNotFound
		}

		return replaceDatabaseAccesses(tx, group.ID, group.DatabaseAccesses)
	})

	if err != nil {
		return domain.Group{}, err
	}

	return group, nil
}

func (r *groupRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Where("group_id = ?", id).
			Delete(&models.UserGroup{}).Error
		if err != nil {
			return err
		}

		err = tx.
			Where("group_id = ?", id).
			Delete(&models.GroupDatabase{}).Error
		if err != nil {
			return err
		}

		result := tx.Delete(&models.Group{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrGroupNotFound
		}

		return nil
	})
}

func (r *groupRepository) AddUser(ctx context.Context, groupID uint, userID uint) error {

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

func (r *groupRepository) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
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

func (r *groupRepository) CountUserGroups(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.
		WithContext(ctx).
		Model(&models.UserGroup{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}

func (r *groupRepository) ListUserIDs(ctx context.Context, groupID uint) ([]uint, error) {
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

func (r *groupRepository) ListAccessTypes(ctx context.Context, userID uint, databaseID uint) ([]domain.AccessType, error) {
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

func replaceDatabaseAccesses(tx *gorm.DB, groupID uint, accesses []domain.DatabaseAccess) error {
	err := tx.
		Where("group_id = ?", groupID).
		Delete(&models.GroupDatabase{}).Error
	if err != nil {
		return err
	}
	if len(accesses) == 0 {
		return nil
	}

	relations := make([]models.GroupDatabase, 0, len(accesses))

	for _, access := range accesses {
		relations = append(relations, models.GroupDatabase{
			GroupID:    groupID,
			DatabaseID: access.DatabaseID,
			AccessType: string(access.AccessType),
		})
	}

	return tx.Create(&relations).Error
}

func (r *groupRepository) ListUserIDsByDatabase(ctx context.Context, databaseID uint) ([]uint, error) {
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

func (r *groupRepository) HasGroupWithOnlyDatabase(ctx context.Context, databaseID uint) (bool, error) {
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
