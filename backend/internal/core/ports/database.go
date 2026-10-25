package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type DatabaseInput = domain.CreateDatabaseInput

type DatabaseService interface {
	GetDatabaseByID(ctx context.Context, databaseID uint) (*domain.Database, error)
	GetAllDatabases(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error)
	GetDatabaseHosts(ctx context.Context) ([]domain.DatabaseHost, error)
	CreateDatabase(ctx context.Context, database *DatabaseInput) (*domain.Database, error)
	DeleteDatabase(ctx context.Context, databaseID uint) error
	GetDatabasesByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error)
	GetDatabaseByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error)
}

type DatabaseRepository interface {
	FindById(ctx context.Context, databaseID uint) (*domain.Database, error)
	FindAll(ctx context.Context, params pagination.PaginationParams) ([]*domain.Database, int64, error)
	Save(ctx context.Context, database *domain.Database) (*domain.Database, error)
	Delete(ctx context.Context, databaseID uint) error
	FindAllByUserID(ctx context.Context, userID uint, params pagination.PaginationParams) ([]*domain.Database, int64, error)
	FindByIDAndUserID(ctx context.Context, databaseID uint, userID uint) (*domain.Database, error)

	FindWithCredentialByID(ctx context.Context, databaseID uint) (domain.Database, error)
	FindCredentialByHostAndPort(ctx context.Context, host string, port uint, user string) (string, error)
	GetDistinctHosts(ctx context.Context) ([]domain.DatabaseHost, error)
	ExistsAll(ctx context.Context, databaseIDs []uint) (bool, error)
	CreateDatabase(ctx context.Context, database *domain.Database) (*domain.Database, error)
}

type DatabaseCredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type DatabaseConnectionTester interface {
	Test(ctx context.Context, database domain.Database) error
}

type DatabasePoolInvalidator interface {
	Invalidate(databaseID uint) error
}
