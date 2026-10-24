package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type DatabaseService interface {
}

type DatabaseRepository interface {
	List(ctx context.Context) ([]domain.Database, error)
	FindByID(ctx context.Context, id uint) (domain.Database, error)
	FindWithCredentialByID(ctx context.Context, id uint) (domain.Database, error)
	ExistsAll(ctx context.Context, ids []uint) (bool, error)
	Create(ctx context.Context, database domain.Database) (domain.Database, error)
	Delete(ctx context.Context, id uint) error
}

type DatabaseCredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}