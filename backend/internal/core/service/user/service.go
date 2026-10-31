package user

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	users      ports.UserRepository
	groups     ports.GroupRepository
	sessions   ports.SessionService
	userGroups ports.UserGroupRepository
}

var _ ports.UserService = (*userService)(nil)

func NewUserService(users ports.UserRepository, groups ports.GroupRepository, sessions ports.SessionService, userGroups ports.UserGroupRepository) ports.UserService {
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

	if username == "" ||
		input.Password == "" ||
		len(input.GroupIDs) == 0 ||
		len([]byte(input.Password)) > 72 {
		return domain.User{}, domain.ErrInvalidInput
	}

	exists, err := s.groups.AllExist(ctx, input.GroupIDs)
	if err != nil {
		return domain.User{}, err
	}

	if !exists {
		return domain.User{}, domain.ErrInvalidGroupsIDs
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return domain.User{}, err
	}

	return s.users.Save(
		ctx,
		domain.User{
			Username: username,
			Password: string(passwordHash),
			Role:     domain.RoleUser,
		},
		input.GroupIDs,
	)
}

func (s *userService) UpdateUser(ctx context.Context, id uint, input domain.UpdateUserInput) (domain.User, error) {
	if _, err := s.users.FindByID(ctx, id); err != nil {
		return domain.User{}, err
	}

	var username *string

	if input.Username != nil {
		trimmed := strings.TrimSpace(*input.Username)

		if trimmed == "" {
			return domain.User{}, domain.ErrInvalidInput
		}

		username = &trimmed
	}

	var passwordHash *string

	if input.Password != nil {
		if *input.Password == "" || len([]byte(*input.Password)) > 72 {
			return domain.User{}, domain.ErrInvalidInput
		}

		hash, err := bcrypt.GenerateFromPassword(
			[]byte(*input.Password),
			bcrypt.DefaultCost,
		)

		if err != nil {
			return domain.User{}, err
		}

		hashed := string(hash)
		passwordHash = &hashed
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
		adminCount, err := s.users.CountByRole(ctx, domain.RoleAdmin)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return domain.ErrLastAdmin
		}
	}

	if err := s.users.Delete(ctx, id); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, id)
}

func (s *userService) UpdateUserRole(ctx context.Context, id uint, role string) (domain.User, error) {
	role = strings.ToLower(strings.TrimSpace(role))

	if role != domain.RoleUser && role != domain.RoleAdmin {
		return domain.User{}, domain.ErrInvalidRole
	}

	current, err := s.users.FindByID(ctx, id)

	if err != nil {
		return domain.User{}, err
	}

	if current.Role == role {
		return current, nil
	}

	if role == domain.RoleUser {
		var groupCount int64
		if s.userGroups != nil {
			var err error
			groupCount, err = s.userGroups.CountByUserID(ctx, id)
			if err != nil {
				return domain.User{}, err
			}
		}

		if groupCount == 0 {
			return domain.User{}, domain.ErrUserMustHaveGroup
		}

		if current.Role == domain.RoleAdmin {
			adminCount, err := s.users.CountByRole(ctx, domain.RoleAdmin)
			if err != nil {
				return domain.User{}, err
			}
			if adminCount <= 1 {
				return domain.User{}, domain.ErrLastAdmin
			}
		}
	}

	updated, err := s.users.UpdateRole(ctx, id, role)

	if err != nil {
		return domain.User{}, err
	}

	if err := s.sessions.RevokeUser(ctx, id); err != nil {
		return domain.User{}, err
	}

	return updated, nil
}
