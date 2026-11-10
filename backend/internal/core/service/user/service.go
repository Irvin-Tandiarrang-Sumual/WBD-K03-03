package user

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/hasher"
)

type userService struct {
	users      ports.UserRepository
	groups     ports.GroupRepository
	sessions   ports.SessionService
	userGroups ports.UserGroupRepository
}

var _ ports.UserService = (*userService)(nil)

func NewUserService(
	users ports.UserRepository,
	groups ports.GroupRepository,
	sessions ports.SessionService,
	userGroups ports.UserGroupRepository,
) ports.UserService {
	return &userService{
		users:      users,
		groups:     groups,
		sessions:   sessions,
		userGroups: userGroups,
	}
}

func (s *userService) GetAllUsers(ctx context.Context, params pagination.PaginationParams) ([]domain.User, int64, error) {
	return s.users.FindAll(ctx, params)
}

func (s *userService) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *userService) CreateUser(ctx context.Context, input domain.CreateUserInput) (domain.User, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || len(username) > domain.MaxUsernameLength || len(input.GroupIDs) == 0 {
		return domain.User{}, domain.ErrInvalidInput
	}
	if err := domain.ValidatePassword(input.Password); err != nil {
		return domain.User{}, err
	}

	if err := s.ensureGroupsExist(ctx, input.GroupIDs); err != nil {
		return domain.User{}, err
	}

	passwordHash, err := hasher.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}

	return s.users.Save(
		ctx,
		domain.User{
			Username: username,
			Password: passwordHash,
			Role:     domain.RoleUser,
		},
		input.GroupIDs,
	)
}

func (s *userService) UpdateUser(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
	if _, err := s.users.FindByID(ctx, id); err != nil {
		return domain.User{}, err
	}

	username, err := resolveUpdatedUsername(input.Username)
	if err != nil {
		return domain.User{}, err
	}

	passwordHash, err := resolveUpdatedPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}

	updated, err := s.users.Update(ctx, id, username, passwordHash)
	if err != nil {
		return domain.User{}, err
	}

	if passwordHash != nil {
		if err := s.sessions.RevokeUser(ctx, id); err != nil {
			return domain.User{}, err
		}
	}

	return updated, nil
}

func (s *userService) DeleteUser(ctx context.Context, id uint) error {
	current, err := s.users.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if current.Role == domain.RoleAdmin {
		if err := s.ensureNotLastAdmin(ctx); err != nil {
			return err
		}
	}

	if err := s.users.Delete(ctx, id); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, id)
}

func (s *userService) UpdateUserRole(ctx context.Context, id uint, role string) (domain.User, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if !domain.IsValidRole(normalizedRole) {
		return domain.User{}, domain.ErrInvalidRole
	}

	current, err := s.users.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	if current.Role == normalizedRole {
		return current, nil
	}

	if normalizedRole == domain.RoleUser {
		if err := s.ensureCanDemoteToUser(ctx, id, current.Role); err != nil {
			return domain.User{}, err
		}
	}

	updated, err := s.users.UpdateRole(ctx, id, normalizedRole)
	if err != nil {
		return domain.User{}, err
	}

	if err := s.sessions.RevokeUser(ctx, id); err != nil {
		return domain.User{}, err
	}

	return updated, nil
}

func (s *userService) ensureGroupsExist(ctx context.Context, groupIDs []uint) error {
	exists, err := s.groups.AllExist(ctx, groupIDs)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrInvalidGroupsIDs
	}
	return nil
}

func (s *userService) ensureCanDemoteToUser(ctx context.Context, userID uint, _ string) error {
	if err := s.ensureHasGroupMembership(ctx, userID); err != nil {
		return err
	}
	return s.ensureNotLastAdmin(ctx)
}

func (s *userService) ensureHasGroupMembership(ctx context.Context, userID uint) error {
	if s.userGroups == nil {
		return domain.ErrUserMustHaveGroup
	}
	groupCount, err := s.userGroups.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if groupCount == 0 {
		return domain.ErrUserMustHaveGroup
	}
	return nil
}

func (s *userService) ensureNotLastAdmin(ctx context.Context) error {
	adminCount, err := s.users.CountByRole(ctx, domain.RoleAdmin)
	if err != nil {
		return err
	}
	if adminCount <= 1 {
		return domain.ErrLastAdmin
	}
	return nil
}

func resolveUpdatedUsername(input *string) (*string, error) {
	if input == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*input)
	if trimmed == "" || len(trimmed) > domain.MaxUsernameLength {
		return nil, domain.ErrInvalidInput
	}
	return &trimmed, nil
}

func resolveUpdatedPassword(input *string) (*string, error) {
	if input == nil {
		return nil, nil
	}
	if err := domain.ValidatePassword(*input); err != nil {
		return nil, err
	}
	hashed, err := hasher.HashPassword(*input)
	if err != nil {
		return nil, err
	}
	return &hashed, nil
}

