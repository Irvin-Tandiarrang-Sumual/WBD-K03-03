package database

import (
	"context"
	"errors"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
)

type databaseRepository struct {
	db     *gorm.DB
	cipher ports.DatabaseCredentialCipher
}

func NewDatabaseRepository(db *gorm.DB, cipher ports.DatabaseCredentialCipher) ports.DatabaseRepository {
	return &databaseRepository{
		db:     db,
		cipher: cipher,
	}
}

func (r *databaseRepository) FindById(ctx context.Context, databaseID uint) (*domain.Database, error) {
	database, err := r.FindByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}
	return &database, nil
}

func (r *databaseRepository) FindAll(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Database{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storedDatabases []model.Database
	if err := query.Select("id", "port", "host", "user", "type", "name").
		Order("id ASC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&storedDatabases).Error; err != nil {
		return nil, 0, err
	}

	databases := make([]*domain.Database, 0, len(storedDatabases))
	for _, storedDatabase := range storedDatabases {
		database := toDomain(storedDatabase, "")
		databases = append(databases, &database)
	}
	return databases, total, nil
}

func (r *databaseRepository) Save(ctx context.Context, database *domain.Database) (*domain.Database, error) {
	if database == nil {
		return nil, domain.ErrInvalidDatabaseInput
	}
	return r.CreateDatabase(ctx, database)
}

func (r *databaseRepository) FindAllByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Database{}).
		Joins("JOIN group_databases ON group_databases.database_id = databases.id").
		Joins("JOIN user_groups ON user_groups.group_id = group_databases.group_id").
		Where("user_groups.user_id = ?", userID)

	var total int64
	if err := query.Distinct("databases.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storedDatabases []model.Database
	if err := query.Select("databases.*").Distinct().
		Order("databases.id ASC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&storedDatabases).Error; err != nil {
		return nil, 0, err
	}

	databaseIDs := make([]uint, 0, len(storedDatabases))
	for _, storedDatabase := range storedDatabases {
		databaseIDs = append(databaseIDs, storedDatabase.ID)
	}

	accessByDatabase, err := r.effectiveAccessByDatabase(ctx, userID, databaseIDs)
	if err != nil {
		return nil, 0, err
	}

	databases := make([]*domain.Database, 0, len(storedDatabases))
	for _, storedDatabase := range storedDatabases {
		database := toDomain(storedDatabase, "")
		database.AccessType = accessByDatabase[storedDatabase.ID]
		databases = append(databases, &database)
	}
	return databases, total, nil
}

func (r *databaseRepository) FindByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error) {
	var storedDatabase model.Database
	err := r.db.WithContext(ctx).
		Model(&model.Database{}).
		Joins("JOIN group_databases ON group_databases.database_id = databases.id").
		Joins("JOIN user_groups ON user_groups.group_id = group_databases.group_id").
		Where("databases.id = ? AND user_groups.user_id = ?", databaseID, userID).
		Select("databases.id", "databases.port", "databases.host", "databases.user", "databases.type", "databases.name").
		First(&storedDatabase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDatabaseNotFound
		}
		return nil, err
	}

	accessByDatabase, err := r.effectiveAccessByDatabase(ctx, userID, []uint{storedDatabase.ID})
	if err != nil {
		return nil, err
	}

	database := toDomain(storedDatabase, "")
	database.AccessType = accessByDatabase[storedDatabase.ID]
	return &database, nil
}

func (r *databaseRepository) effectiveAccessByDatabase(ctx context.Context, userID uint, databaseIDs []uint) (map[uint]string, error) {
	accessMap := make(map[uint]string, len(databaseIDs))
	if len(databaseIDs) == 0 {
		return accessMap, nil
	}

	var rows []struct {
		DatabaseID uint
		AccessType string
	}
	err := r.db.WithContext(ctx).
		Table("group_databases AS gd").
		Select("gd.database_id, gd.access_type").
		Joins("JOIN user_groups AS ug ON ug.group_id = gd.group_id").
		Where("ug.user_id = ? AND gd.database_id IN ?", userID, databaseIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.AccessType == string(domain.AccessViewEdit) {
			accessMap[row.DatabaseID] = string(domain.AccessViewEdit)
		} else if accessMap[row.DatabaseID] == "" {
			accessMap[row.DatabaseID] = row.AccessType
		}
	}

	return accessMap, nil
}

func (r *databaseRepository) FindByID(ctx context.Context, databaseID uint) (domain.Database, error) {
	var storedDatabase model.Database
	err := r.db.WithContext(ctx).
		Select("id", "port", "host", "user", "type", "name").
		First(&storedDatabase, databaseID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Database{}, domain.ErrDatabaseNotFound
		}
		return domain.Database{}, err
	}
	return toDomain(storedDatabase, ""), nil
}

func (r *databaseRepository) FindWithCredentialByID(ctx context.Context, databaseID uint) (domain.Database, error) {
	var storedDatabase model.Database
	if err := r.db.WithContext(ctx).First(&storedDatabase, databaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Database{}, domain.ErrDatabaseNotFound
		}
		return domain.Database{}, err
	}

	password, err := r.cipher.Decrypt(storedDatabase.Password)
	if err != nil {
		return domain.Database{}, err
	}
	return toDomain(storedDatabase, password), nil
}

func (r *databaseRepository) ExistsAll(ctx context.Context, databaseIDs []uint) (bool, error) {
	if len(databaseIDs) == 0 {
		return true, nil
	}

	uniqueIDs := make(map[uint]struct{}, len(databaseIDs))
	for _, databaseID := range databaseIDs {
		uniqueIDs[databaseID] = struct{}{}
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Database{}).
		Where("id IN ?", databaseIDs).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == int64(len(uniqueIDs)), nil
}

func (r *databaseRepository) CreateDatabase(ctx context.Context, database *domain.Database) (*domain.Database, error) {
	if database == nil {
		return nil, domain.ErrInvalidDatabaseInput
	}

	encryptedPassword, err := r.cipher.Encrypt(database.Password)
	if err != nil {
		return nil, err
	}

	storedDatabase := model.Database{
		Port:     database.Port,
		Host:     database.Host,
		User:     database.User,
		Password: encryptedPassword,
		Type:     database.Type,
		Name:     database.Name,
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&storedDatabase).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrDatabaseAlreadyExists
			}
			return err
		}

		var adminGroup model.Group
		if err := tx.Where("name = ?", "admin").First(&adminGroup).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		return tx.Create(&model.GroupDatabase{
			GroupID:    adminGroup.ID,
			DatabaseID: storedDatabase.ID,
			AccessType: string(domain.AccessViewEdit),
		}).Error
	})
	if err != nil {
		return nil, err
	}

	createdDatabase := *database
	createdDatabase.ID = storedDatabase.ID
	createdDatabase.Password = ""
	return &createdDatabase, nil
}

func (r *databaseRepository) Delete(ctx context.Context, databaseID uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Database{}, databaseID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrDatabaseNotFound
	}
	return nil
}

func toDomain(database model.Database, password string) domain.Database {
	return domain.Database{
		ID:       database.ID,
		Port:     database.Port,
		Host:     database.Host,
		User:     database.User,
		Password: password,
		Type:     database.Type,
		Name:     database.Name,
	}
}
