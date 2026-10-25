package session

import (
	"context"
	"errors"
	"time"

	models "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

var _ ports.SessionRepository = (*sessionRepository)(nil)

func NewSessionRepository(db *gorm.DB) ports.SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) Create(ctx context.Context, session domain.Session) (domain.Session, error) {
	rotatedAt := session.RotatedAt
	if rotatedAt.IsZero() {
		rotatedAt = session.LastActive
	}

	var gracePeriodUntil *time.Time
	if !session.GracePeriodUntil.IsZero() {
		t := session.GracePeriodUntil
		gracePeriodUntil = &t
	}

	storedSession := models.Session{
		OpaqueToken:         session.OpaqueToken,
		PreviousOpaqueToken: session.PreviousOpaqueToken,
		GracePeriodUntil:    gracePeriodUntil,
		RotatedAt:           rotatedAt,
		IdleTimeout:         session.IdleTimeout,
		LastActive:          session.LastActive,
		AbsoluteExpiration:  session.AbsoluteExpiration,
		UserID:              session.UserID,
	}

	if err := r.db.
		WithContext(ctx).
		Create(&storedSession).Error; err != nil {
		return domain.Session{}, err
	}

	session.ID = storedSession.ID

	return session, nil
}

func (r *sessionRepository) FindByToken(ctx context.Context, opaqueToken string) (domain.Session, error) {
	var storedSession models.Session

	err := r.db.
		WithContext(ctx).
		Where("opaque_token = ? OR (previous_opaque_token = ? AND previous_opaque_token <> '')", opaqueToken, opaqueToken).
		First(&storedSession).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Session{}, domain.ErrInvalidSession
		}

		return domain.Session{}, err
	}

	return toDomain(storedSession), nil
}

func (r *sessionRepository) Touch(ctx context.Context, id uint, lastActive time.Time) error {
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

func (r *sessionRepository) Rotate(ctx context.Context, session domain.Session) error {
	var gracePeriodUntil *time.Time
	if !session.GracePeriodUntil.IsZero() {
		t := session.GracePeriodUntil
		gracePeriodUntil = &t
	}

	result := r.db.
		WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", session.ID).
		Updates(map[string]any{
			"opaque_token":          session.OpaqueToken,
			"previous_opaque_token": session.PreviousOpaqueToken,
			"grace_period_until":    gracePeriodUntil,
			"rotated_at":            session.RotatedAt,
			"last_active":           session.LastActive,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrInvalidSession
	}

	return nil
}

func (r *sessionRepository) DeleteByToken(ctx context.Context, opaqueToken string) error {
	return r.db.
		WithContext(ctx).
		Where("opaque_token = ? OR (previous_opaque_token = ? AND previous_opaque_token <> '')", opaqueToken, opaqueToken).
		Delete(&models.Session{}).Error
}

// DeleteByUserID mencabut seluruh session seorang user.
func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.Session{}).Error
}

func toDomain(storedSession models.Session) domain.Session {
	var gracePeriodUntil time.Time
	if storedSession.GracePeriodUntil != nil {
		gracePeriodUntil = *storedSession.GracePeriodUntil
	}

	return domain.Session{
		ID:                  storedSession.ID,
		OpaqueToken:         storedSession.OpaqueToken,
		PreviousOpaqueToken: storedSession.PreviousOpaqueToken,
		GracePeriodUntil:    gracePeriodUntil,
		RotatedAt:           storedSession.RotatedAt,
		IdleTimeout:         storedSession.IdleTimeout,
		LastActive:          storedSession.LastActive,
		AbsoluteExpiration:  storedSession.AbsoluteExpiration,
		UserID:              storedSession.UserID,
	}
}
