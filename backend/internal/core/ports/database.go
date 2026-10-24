package ports

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type DatabaseService interface {
}

type DatabaseRepository interface {
	ExistsAll(ctx context.Context, ids []uint) (bool, error)

	FindByID(ctx context.Context, id uint) (domain.Database, error)
}

type DatabaseCredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}