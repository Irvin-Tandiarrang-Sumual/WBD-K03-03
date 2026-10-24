package session

import (
	"context"
	"errors"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations/models"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

var _ ports.SessionRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, session domain.Session) (domain.Session, error) {
	storedSession := models.Session{
		OpaqueToken:        session.OpaqueToken,
		IdleTimeout:        session.IdleTimeout,
		LastActive:         session.LastActive,
		AbsoluteExpiration: session.AbsoluteExpiration,
		UserID:             session.UserID,
	}

	if err := r.db.
		WithContext(ctx).
		Create(&storedSession).Error; err != nil {
		return domain.Session{}, err
	}

	session.ID = storedSession.ID

	return session, nil
}

func (r *Repository) FindByToken(ctx context.Context, opaqueToken string) (domain.Session, error) {
	var storedSession models.Session

	err := r.db.
		WithContext(ctx).
		Where("opaque_token = ?", opaqueToken).
		First(&storedSession).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Session{}, domain.ErrInvalidSession
		}

		return domain.Session{}, err
	}

	return toDomain(storedSession), nil
}

func (r *Repository) Touch(ctx context.Context, id uint, lastActive time.Time) error {
	result := r.db.
		WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", id).
		Update("last_active", lastActive)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrInvalidSession
	}

	return nil
}

func (r *Repository) DeleteByToken(ctx context.Context, opaqueToken string) error {
	return r.db.
		WithContext(ctx).
		Where("opaque_token = ?", opaqueToken).
		Delete(&models.Session{}).Error
}

// DeleteByUserID mencabut seluruh session seorang user.
func (r *Repository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.Session{}).Error
}

func toDomain(storedSession models.Session) domain.Session {
	return domain.Session{
		ID:                 storedSession.ID,
		OpaqueToken:        storedSession.OpaqueToken,
		IdleTimeout:        storedSession.IdleTimeout,
		LastActive:         storedSession.LastActive,
		AbsoluteExpiration: storedSession.AbsoluteExpiration,
		UserID:             storedSession.UserID,
	}
}