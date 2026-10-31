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
	userGroups ports.UserGroupRepository,
	groups ports.GroupRepository,
	users ports.UserRepository,
	sessions ports.SessionService,
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

	memberIDs, err := s.userGroups.FindAllUserIDsByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	if !containsUserID(memberIDs, userID) {
		return domain.ErrMembershipNotFound
	}

	groupCount, err := s.userGroups.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Role == "user" && groupCount <= 1 {
		return domain.ErrUserMustHaveGroup
	}

	if err := s.userGroups.RemoveUser(ctx, groupID, userID); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, userID)
}

func (s *userGroupService) GetEffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	accesses, err := s.userGroups.FindAllAccessTypes(ctx, userID, databaseID)
	if err != nil {
		return "", err
	}

	effective := domain.AccessType("")

	for _, access := range accesses {
		if access == domain.AccessViewEdit {
			return domain.AccessViewEdit, nil
		}
		if access == domain.AccessViewOnly {
			effective = domain.AccessViewOnly
		}
	}

	if effective == "" {
		return "", domain.ErrDatabaseAccessDenied
	}

	return effective, nil
}

func (s *userGroupService) EffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	return s.GetEffectiveDatabaseAccess(ctx, userID, databaseID)
}

func containsUserID(userIDs []uint, target uint) bool {
	for _, userID := range userIDs {
		if userID == target {
			return true
		}
	}
	return false
}
