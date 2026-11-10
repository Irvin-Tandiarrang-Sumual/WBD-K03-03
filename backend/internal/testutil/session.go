package testutil

import (
	"context"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type SessionRepository struct {
	ports.SessionRepository
	SaveFunc            func(context.Context, domain.Session) (domain.Session, error)
	FindByTokenFunc     func(context.Context, string) (domain.Session, error)
	TouchFunc           func(context.Context, uint, time.Time) error
	RotateFunc          func(context.Context, domain.Session) (bool, error)
	DeleteByTokenFunc   func(context.Context, string) error
	DeleteByUserIDFunc  func(context.Context, uint) error
	DeleteByUserIDsFunc func(context.Context, []uint) error
}

func (f *SessionRepository) Save(ctx context.Context, session domain.Session) (domain.Session, error) {
	if f.SaveFunc != nil {
		return f.SaveFunc(ctx, session)
	}
	return domain.Session{}, nil
}

func (f *SessionRepository) FindByToken(ctx context.Context, token string) (domain.Session, error) {
	if f.FindByTokenFunc != nil {
		return f.FindByTokenFunc(ctx, token)
	}
	return domain.Session{}, nil
}

func (f *SessionRepository) Touch(ctx context.Context, id uint, lastActive time.Time) error {
	if f.TouchFunc != nil {
		return f.TouchFunc(ctx, id, lastActive)
	}
	return nil
}

func (f *SessionRepository) Rotate(ctx context.Context, session domain.Session) (bool, error) {
	if f.RotateFunc != nil {
		return f.RotateFunc(ctx, session)
	}
	return true, nil
}

func (f *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	if f.DeleteByTokenFunc != nil {
		return f.DeleteByTokenFunc(ctx, token)
	}
	return nil
}

func (f *SessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	if f.DeleteByUserIDFunc != nil {
		return f.DeleteByUserIDFunc(ctx, userID)
	}
	return nil
}

func (f *SessionRepository) DeleteByUserIDs(ctx context.Context, userIDs []uint) error {
	if f.DeleteByUserIDsFunc != nil {
		return f.DeleteByUserIDsFunc(ctx, userIDs)
	}
	return nil
}

type SessionService struct {
	ports.SessionService
	CreateSessionFunc func(context.Context, uint) (domain.Session, error)
	ResolveFunc       func(context.Context, string) (domain.Session, error)
	RevokeFunc        func(context.Context, string) error
	RevokeUserFunc    func(context.Context, uint) error
	RevokeUsersFunc   func(context.Context, []uint) error
}

func (f *SessionService) CreateSession(ctx context.Context, userID uint) (domain.Session, error) {
	if f.CreateSessionFunc != nil {
		return f.CreateSessionFunc(ctx, userID)
	}
	return domain.Session{}, nil
}

func (f *SessionService) Resolve(ctx context.Context, token string) (domain.Session, error) {
	if f.ResolveFunc != nil {
		return f.ResolveFunc(ctx, token)
	}
	return domain.Session{}, nil
}

func (f *SessionService) Revoke(ctx context.Context, token string) error {
	if f.RevokeFunc != nil {
		return f.RevokeFunc(ctx, token)
	}
	return nil
}

func (f *SessionService) RevokeUser(ctx context.Context, userID uint) error {
	if f.RevokeUserFunc != nil {
		return f.RevokeUserFunc(ctx, userID)
	}
	return nil
}

func (f *SessionService) RevokeUsers(ctx context.Context, userIDs []uint) error {
	if f.RevokeUsersFunc != nil {
		return f.RevokeUsersFunc(ctx, userIDs)
	}
	return nil
}
