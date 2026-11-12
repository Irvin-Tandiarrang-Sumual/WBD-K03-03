package usergroup

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type userGroupService struct {
	userGroups ports.UserGroupRepository
	groups     ports.GroupRepository
	users      ports.UserRepository
	sessions   ports.SessionService
}

var _ ports.UserGroupService = (*userGroupService)(nil)

func NewUserGroupService(
	userGroups ports.UserGroupRepository, groups ports.GroupRepository,
	users ports.UserRepository, sessions ports.SessionService,
) ports.UserGroupService {
	return &userGroupService{
		userGroups: userGroups,
		groups:     groups,
		users:      users,
		sessions:   sessions,
	}
}

func (s *userGroupService) AddUser(ctx context.Context, groupID uint, userID uint) error {
	if _, err := s.groups.FindByID(ctx, groupID); err != nil {
		return err
	}
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return err
	}
	if err := s.userGroups.AddUser(ctx, groupID, userID); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, userID)
}

func (s *userGroupService) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	if _, err := s.groups.FindByID(ctx, groupID); err != nil {
		return err
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.ensureCanRemoveUser(ctx, groupID, user); err != nil {
		return err
	}

	if err := s.userGroups.RemoveUser(ctx, groupID, user.ID); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, user.ID)
}

func (s *userGroupService) Authorize(
	ctx context.Context, userID uint,
	databaseID uint, requiredAccessType domain.AccessType,
) error {
	accesses, err := s.userGroups.FindAllAccessTypes(ctx, userID, databaseID)
	if err != nil {
		return err
	}

	for _, access := range accesses {
		if access.Allows(requiredAccessType) {
			return nil
		}
	}

	return domain.ErrDatabaseAccessDenied
}

func (s *userGroupService) ensureCanRemoveUser(ctx context.Context, groupID uint, user domain.User) error {
	memberIDs, err := s.userGroups.FindAllUserIDsByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	if !containsUserID(memberIDs, user.ID) {
		return domain.ErrMembershipNotFound
	}

	groupCount, err := s.userGroups.CountByUserID(ctx, user.ID)
	if err != nil {
		return err
	}

	if user.Role == domain.RoleUser && groupCount <= 1 {
		return domain.ErrUserMustHaveGroup
	}

	return nil
}

func containsUserID(userIDs []uint, target uint) bool {
	for _, userID := range userIDs {
		if userID == target {
			return true
		}
	}
	return false
}
