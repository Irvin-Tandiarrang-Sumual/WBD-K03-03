package repositories

import (
	"context"
	"errors"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"gorm.io/gorm"
)

type migrationRepository struct {
	db *gorm.DB
}

func NewMigrationRepository(db *gorm.DB) ports.MigrationRepository {
	return &migrationRepository{db: db}
}

func (r *migrationRepository) FindAllByDatabaseID(ctx context.Context, databaseID uint, params pagination.PaginationParams) ([]*domain.Migration, int64, error) {
	query := r.db.WithContext(ctx).Where("database_id = ?", databaseID)

	var total int64
	if err := query.Model(&model.Migration{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var migrations []model.Migration
	if err := query.Order("id ASC").
		Offset(params.Offset()).
		Limit(params.Limit).
		Find(&migrations).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*domain.Migration, 0, len(migrations))
	for _, migration := range migrations {
		result = append(result, toDomainMigration(migration))
	}
	return result, total, nil
}

func (r *migrationRepository) FindLatestByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error) {
	var migration model.Migration
	err := r.db.WithContext(ctx).
		Where("database_id = ?", databaseID).
		Order("id DESC").
		First(&migration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMigrationNotFound
		}
		return nil, err
	}
	return toDomainMigration(migration), nil
}

func (r *migrationRepository) FindLatestSuccessfulByDatabaseID(ctx context.Context, databaseID uint) (*domain.Migration, error) {
	var migration model.Migration
	err := r.db.WithContext(ctx).
		Where("database_id = ? AND status = ?", databaseID, "success").
		Order("id DESC").
		First(&migration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMigrationNotFound
		}
		return nil, err
	}
	return toDomainMigration(migration), nil
}

func (r *migrationRepository) Save(ctx context.Context, migration *domain.Migration) (*domain.Migration, error) {
	if migration == nil {
		return nil, domain.ErrInvalidMigrationInput
	}

	migrationModel := model.Migration{
		ID:           migration.ID,
		DatabaseID:   int(migration.DatabaseID),
		Version:      migration.Version,
		MigratedAt:   migration.MigratedAt,
		Status:       migration.Status,
		SqlUp:        migration.SqlUp,
		SqlDown:      migration.SqlDown,
		ErrorMessage: migration.ErrorMessage,
		Snapshot:     migration.Snapshot,
		UserID:       int(migration.UserID),
	}
	if err := r.db.WithContext(ctx).Create(&migrationModel).Error; err != nil {
		return nil, err
	}
	return toDomainMigration(migrationModel), nil
}

func (r *migrationRepository) Update(ctx context.Context, migration *domain.Migration) (*domain.Migration, error) {
	if migration == nil || migration.ID == 0 {
		return nil, domain.ErrInvalidMigrationInput
	}

	migrationModel := toModel(migration)
	if err := r.db.WithContext(ctx).Save(&migrationModel).Error; err != nil {
		return nil, err
	}
	return toDomainMigration(migrationModel), nil
}

func (r *migrationRepository) FindByIDAndDatabaseID(ctx context.Context, migrationID uint, databaseID uint) (*domain.Migration, error) {
	var migration model.Migration
	err := r.db.WithContext(ctx).
		Where("id = ? AND database_id = ?", migrationID, databaseID).
		First(&migration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMigrationNotFound
		}
		return nil, err
	}
	return toDomainMigration(migration), nil
}

func (r *migrationRepository) FindByVersionAndDatabaseID(ctx context.Context, version string, databaseID uint) (*domain.Migration, error) {
	var migration model.Migration
	err := r.db.WithContext(ctx).
		Where("version = ? AND database_id = ?", version, databaseID).
		First(&migration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMigrationNotFound
		}
		return nil, err
	}
	return toDomainMigration(migration), nil
}

func toDomainMigration(migration model.Migration) *domain.Migration {
	return &domain.Migration{
		ID:           migration.ID,
		DatabaseID:   uint(migration.DatabaseID),
		Version:      migration.Version,
		MigratedAt:   migration.MigratedAt,
		Status:       migration.Status,
		SqlUp:        migration.SqlUp,
		SqlDown:      migration.SqlDown,
		ErrorMessage: migration.ErrorMessage,
		Snapshot:     []byte(migration.Snapshot),
		UserID:       uint(migration.UserID),
	}
}

func toModel(migration *domain.Migration) model.Migration {
	return model.Migration{
		ID:           migration.ID,
		DatabaseID:   int(migration.DatabaseID),
		Version:      migration.Version,
		MigratedAt:   migration.MigratedAt,
		Status:       migration.Status,
		SqlUp:        migration.SqlUp,
		SqlDown:      migration.SqlDown,
		ErrorMessage: migration.ErrorMessage,
		Snapshot:     migration.Snapshot,
		UserID:       int(migration.UserID),
	}
}
