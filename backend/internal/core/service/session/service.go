package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

const (
	DefaultIdleTimeout         = 30 * time.Minute
	DefaultAbsoluteLifetime    = 24 * time.Hour
	DefaultRotationInterval    = 15 * time.Minute
	DefaultRotationGracePeriod = 30 * time.Second

	sessionTokenBytes = 32
)

type sessionService struct {
	sessions ports.SessionRepository
}

var _ ports.SessionService = (*sessionService)(nil)

func NewSessionService(sessions ports.SessionRepository) ports.SessionService {
	return &sessionService{
		sessions: sessions,
	}
}

func (s *sessionService) CreateSession(ctx context.Context, userID uint) (domain.Session, error) {
	opaqueToken, err := generateOpaqueToken()
	if err != nil {
		return domain.Session{}, err
	}

	now := time.Now().UTC()

	session := domain.Session{
		OpaqueToken:        opaqueToken,
		RotatedAt:          now,
		IdleTimeout:        DefaultIdleTimeout,
		LastActive:         now,
		AbsoluteExpiration: now.Add(DefaultAbsoluteLifetime),
		UserID:             userID,
	}

	return s.sessions.Save(ctx, session)
}

func (s *sessionService) Resolve(ctx context.Context, opaqueToken string) (domain.Session, error) {
	opaqueToken = strings.TrimSpace(opaqueToken)

	if opaqueToken == "" {
		return domain.Session{}, domain.ErrInvalidSession
	}

	session, err := s.sessions.FindByToken(ctx, opaqueToken)
	if err != nil {
		return domain.Session{}, err
	}

	now := time.Now().UTC()

	absoluteExpired :=
		!session.AbsoluteExpiration.IsZero() &&
			!now.Before(session.AbsoluteExpiration)

	idleExpired :=
		session.IdleTimeout > 0 &&
			now.Sub(session.LastActive) > session.IdleTimeout

	if absoluteExpired || idleExpired {
		if err := s.sessions.DeleteByToken(ctx, opaqueToken); err != nil {
			return domain.Session{}, err
		}

		return domain.Session{}, domain.ErrInvalidSession
	}

	// Jika request datang menggunakan identifier lama (PreviousOpaqueToken),
	// periksa apakah masih berada dalam masa grace period.
	if session.PreviousOpaqueToken != "" &&
		opaqueToken == session.PreviousOpaqueToken &&
		opaqueToken != session.OpaqueToken {
		if session.GracePeriodUntil.IsZero() || !now.Before(session.GracePeriodUntil) {
			return domain.Session{}, domain.ErrInvalidSession
		}

		if err := s.sessions.Touch(ctx, session.ID, now); err != nil {
			return domain.Session{}, err
		}

		session.LastActive = now
		return session, nil
	}

	// Rotasi berkala apabila umur token saat ini telah mencapai DefaultRotationInterval,
	// tanpa memperpanjang AbsoluteExpiration.
	if !session.RotatedAt.IsZero() && now.Sub(session.RotatedAt) >= DefaultRotationInterval {
		newToken, genErr := generateOpaqueToken()
		if genErr != nil {
			return domain.Session{}, genErr
		}

		session.PreviousOpaqueToken = session.OpaqueToken
		session.OpaqueToken = newToken
		session.GracePeriodUntil = now.Add(DefaultRotationGracePeriod)
		session.RotatedAt = now
		session.LastActive = now

		if err := s.sessions.Rotate(ctx, session); err != nil {
			return domain.Session{}, err
		}

		return session, nil
	}

	if err := s.sessions.Touch(ctx, session.ID, now); err != nil {
		return domain.Session{}, err
	}

	session.LastActive = now

	return session, nil
}

func (s *sessionService) Revoke(ctx context.Context, opaqueToken string) error {
	if strings.TrimSpace(opaqueToken) == "" {
		return nil
	}

	return s.sessions.DeleteByToken(ctx, opaqueToken)
}

func (s *sessionService) RevokeUser(ctx context.Context, userID uint) error {
	return s.sessions.DeleteByUserID(ctx, userID)
}

func generateOpaqueToken() (string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
