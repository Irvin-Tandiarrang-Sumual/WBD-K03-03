package group

import (
	"context"
	"sort"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

type groupService struct {
	userGroups ports.UserGroupRepository
	groups     ports.GroupRepository
	users      ports.UserRepository
	databases  ports.DatabaseRepository
	sessions   ports.SessionService
}

var _ ports.GroupService = (*groupService)(nil)

func NewGroupService(
	userGroups ports.UserGroupRepository,
	groups ports.GroupRepository,
	users ports.UserRepository,
	databases ports.DatabaseRepository,
	sessions ports.SessionService,
) ports.GroupService {
	return &groupService{
		userGroups: userGroups,
		groups:     groups,
		users:      users,
		databases:  databases,
		sessions:   sessions,
	}
}

func (s *groupService) GetAllGroups(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	return s.groups.FindAll(ctx, params)
}

func (s *groupService) GetGroupByID(ctx context.Context, id uint) (domain.Group, error) {
	return s.groups.FindByID(ctx, id)
}

func (s *groupService) CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > domain.MaxGroupNameLength {
		return domain.Group{}, domain.ErrInvalidInput
	}

	accesses, err := normalizeAccesses(input.DatabaseAccesses)
	if err != nil {
		return domain.Group{}, err
	}

	if err := s.ensureDatabasesExist(ctx, accesses); err != nil {
		return domain.Group{}, err
	}

	if input.LeaderID == 0 {
		return domain.Group{}, domain.ErrInvalidInput
	}

	if _, err := s.users.FindByID(ctx, input.LeaderID); err != nil {
		return domain.Group{}, err
	}

	return s.groups.Save(ctx, domain.Group{
		Name:             name,
		LeaderID:         input.LeaderID,
		DatabaseAccesses: accesses,
	})
}

func (s *groupService) UpdateGroup(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {
	current, err := s.groups.FindByID(ctx, id)
	if err != nil {
		return domain.Group{}, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > domain.MaxGroupNameLength {
			return domain.Group{}, domain.ErrInvalidInput
		}
		current.Name = name
	}

	if input.LeaderID != nil {
		if *input.LeaderID == 0 {
			return domain.Group{}, domain.ErrInvalidInput
		}
		if _, err := s.users.FindByID(ctx, *input.LeaderID); err != nil {
			return domain.Group{}, err
		}

		memberIDs, err := s.userGroups.FindAllUserIDsByGroupID(ctx, id)
		if err != nil {
			return domain.Group{}, err
		}

		isMember := false
		for _, memberID := range memberIDs {
			if memberID == *input.LeaderID {
				isMember = true
				break
			}
		}

		if !isMember {
			return domain.Group{}, domain.ErrLeaderMustBeGroupMember
		}

		current.LeaderID = *input.LeaderID
	}

	accessChanged, err := s.applyGroupAccessesUpdate(ctx, &current, input.DatabaseAccesses)
	if err != nil {
		return domain.Group{}, err
	}

	updated, err := s.groups.Update(ctx, current)
	if err != nil {
		return domain.Group{}, err
	}

	if accessChanged {
		userIDs, err := s.userGroups.FindAllUserIDsByGroupID(ctx, id)
		if err != nil {
			return domain.Group{}, err
		}
		if err := s.sessions.RevokeUsers(ctx, userIDs); err != nil {
			return domain.Group{}, err
		}
	}

	return updated, nil
}

func (s *groupService) DeleteGroup(ctx context.Context, id uint) error {
	if _, err := s.groups.FindByID(ctx, id); err != nil {
		return err
	}

	hasOrphan, err := s.userGroups.HasUserWithOnlyThisGroup(ctx, id)
	if err != nil {
		return err
	}
	if hasOrphan {
		return domain.ErrUserMustHaveGroup
	}

	userIDs, err := s.userGroups.FindAllUserIDsByGroupID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.groups.Delete(ctx, id); err != nil {
		return err
	}

	return s.sessions.RevokeUsers(ctx, userIDs)
}

func (s *groupService) ensureDatabasesExist(ctx context.Context, accesses []domain.DatabaseAccess) error {
	exists, err := s.databases.ExistsAll(ctx, databaseIDs(accesses))
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrDatabaseNotFound
	}
	return nil
}

func (s *groupService) applyGroupAccessesUpdate(
	ctx context.Context,
	current *domain.Group,
	inputAccesses *[]domain.DatabaseAccess,
) (bool, error) {
	if inputAccesses == nil {
		return false, nil
	}

	accesses, err := normalizeAccesses(*inputAccesses)
	if err != nil {
		return false, err
	}

	if err := s.ensureDatabasesExist(ctx, accesses); err != nil {
		return false, err
	}

	accessChanged := !sameAccesses(current.DatabaseAccesses, accesses)
	current.DatabaseAccesses = accesses
	return accessChanged, nil
}

func normalizeAccesses(accesses []domain.DatabaseAccess) ([]domain.DatabaseAccess, error) {
	if len(accesses) == 0 {
		return nil, domain.ErrGroupMustHaveDatabase
	}

	seen := make(map[uint]struct{}, len(accesses))
	result := make([]domain.DatabaseAccess, 0, len(accesses))

	for _, access := range accesses {
		if access.DatabaseID == 0 {
			return nil, domain.ErrInvalidDatabaseAccess
		}

		if access.AccessType != domain.AccessViewOnly && access.AccessType != domain.AccessViewEdit {
			return nil, domain.ErrInvalidDatabaseAccess
		}
		if _, exists := seen[access.DatabaseID]; exists {
			return nil, domain.ErrInvalidDatabaseAccess
		}

		seen[access.DatabaseID] = struct{}{}
		result = append(result, access)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DatabaseID < result[j].DatabaseID
	})

	return result, nil
}

func sameAccesses(left []domain.DatabaseAccess, right []domain.DatabaseAccess) bool {
	if len(left) != len(right) {
		return false
	}

	leftMap := make(map[uint]domain.AccessType, len(left))
	for _, access := range left {
		leftMap[access.DatabaseID] = access.AccessType
	}
	for _, access := range right {
		if leftMap[access.DatabaseID] != access.AccessType {
			return false
		}
	}

	return true
}

func databaseIDs(accesses []domain.DatabaseAccess) []uint {
	ids := make([]uint, 0, len(accesses))
	for _, access := range accesses {
		ids = append(ids, access.DatabaseID)
	}
	return ids
}
