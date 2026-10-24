package database

import (
	"context"
	"errors"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations/models"
	"gorm.io/gorm"
)

type Repository struct {
	db     *gorm.DB
	cipher ports.DatabaseCredentialCipher
}

var _ ports.DatabaseRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB, cipher ports.DatabaseCredentialCipher) *Repository {
	return &Repository{
		db:     db,
		cipher: cipher,
	}
}

func (r *Repository) List(ctx context.Context) ([]domain.Database, error) {
	var storedDatabases []models.Database

	err := r.db.
		WithContext(ctx).
		Select("id", "port", "host", "user", "type", "name").
		Order("id ASC").
		Find(&storedDatabases).Error

	if err != nil {
		return nil, err
	}

	databases := make([]domain.Database, 0, len(storedDatabases))

	for _, storedDatabase := range storedDatabases {
		databases = append(databases, toDomain(storedDatabase, ""))
	}

	return databases, nil
}

func (r *Repository) FindByID(ctx context.Context, id uint) (domain.Database, error) {
	var storedDatabase models.Database

	err := r.db.
		WithContext(ctx).
		Select("id", "port", "host", "user", "type", "name").
		First(&storedDatabase, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Database{}, domain.ErrDatabaseNotFound
		}

		return domain.Database{}, err
	}

	return toDomain(storedDatabase, ""), nil
}

func (r *Repository) FindWithCredentialByID(ctx context.Context, id uint) (domain.Database, error) {
	var storedDatabase models.Database

	err := r.db.WithContext(ctx).First(&storedDatabase, id).Error

	if err != nil {
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

func (r *Repository) ExistsAll(ctx context.Context, ids []uint) (bool, error) {
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
		Model(&models.Database{}).
		Where("id IN ?", cleanIDs).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count == int64(len(cleanIDs)), nil
}

func (r *Repository) Create(ctx context.Context, database domain.Database) (domain.Database, error) {
	
	encryptedPassword, err := r.cipher.Encrypt(database.Password)
	if err != nil {
		return domain.Database{}, err
	}

	storedDatabase := models.Database{
		Port:     database.Port,
		Host:     database.Host,
		User:     database.User,
		Password: encryptedPassword,
		Type:     database.Type,
		Name:     database.Name,
	}

	err = r.db.WithContext(ctx).Create(&storedDatabase).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.Database{}, domain.ErrDatabaseAlreadyExists
		}

		return domain.Database{}, err
	}

	database.ID = storedDatabase.ID
	database.Password = ""

	return database, nil
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("database_id = ?", id).Delete(&models.GroupDatabase{}).Error; err != nil {
			return err
		}

		result := tx.Delete(&models.Database{}, id)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return domain.ErrDatabaseNotFound
		}

		return nil
	})
}

func toDomain(stored models.Database, password string) domain.Database {
	return domain.Database{
		ID:       stored.ID,
		Port:     stored.Port,
		Host:     stored.Host,
		User:     stored.User,
		Password: password,
		Type:     stored.Type,
		Name:     stored.Name,
	}
}