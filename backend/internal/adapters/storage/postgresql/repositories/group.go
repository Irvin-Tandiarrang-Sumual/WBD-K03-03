package repositories

import (
	"context"
	"errors"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			LeaderID:         storedGroup.LeaderID,
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
		LeaderID:         storedGroup.LeaderID,
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
			Name:     group.Name,
			LeaderID: group.LeaderID,
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

		if storedGroup.LeaderID != 0 {
			userGroup := models.UserGroup{
				GroupID: storedGroup.ID,
				UserID:  storedGroup.LeaderID,
			}
			if err := tx.Create(&userGroup).Error; err != nil {
				return err
			}
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
		updates := map[string]any{
			"name": group.Name,
		}
		if group.LeaderID != 0 {
			updates["leader_id"] = group.LeaderID
		}

		result := tx.
			Model(&models.Group{}).
			Where("id = ?", group.ID).
			Updates(updates)

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

func (r *groupRepository) CountByLeaderID(ctx context.Context, leaderID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Group{}).Where("leader_id = ?", leaderID).Count(&count).Error
	return count, err
}

func (r *groupRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.ensureNoOrphanedUsersTx(tx, id); err != nil {
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

func (r *groupRepository) ensureNoOrphanedUsersTx(tx *gorm.DB, groupID uint) error {
	var userIDs []uint
	err := tx.
		Model(&models.User{}).
		Where("id IN (?) AND role = ?",
			tx.Model(&models.UserGroup{}).Select("user_id").Where("group_id = ?", groupID),
			domain.RoleUser,
		).
		Order("id ASC").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Pluck("id", &userIDs).Error
	if err != nil {
		return err
	}

	if len(userIDs) == 0 {
		return nil
	}

	var orphanedUserIDs []uint
	err = tx.
		Model(&models.UserGroup{}).
		Select("user_id").
		Where("user_id IN ?", userIDs).
		Group("user_id").
		Having("COUNT(*) <= 1").
		Limit(1).
		Pluck("user_id", &orphanedUserIDs).Error
	if err != nil {
		return err
	}

	if len(orphanedUserIDs) > 0 {
		return domain.ErrUserMustHaveGroup
	}

	return nil
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
