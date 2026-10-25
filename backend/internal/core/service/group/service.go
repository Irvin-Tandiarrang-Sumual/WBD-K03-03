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
	groups    ports.GroupRepository
	users     ports.UserRepository
	databases ports.DatabaseRepository
	sessions  ports.SessionService
}

func NewGroupService(
	groups ports.GroupRepository,
	users ports.UserRepository,
	databases ports.DatabaseRepository,
	sessions ports.SessionService,
) ports.GroupService {
	return &groupService{
		groups:    groups,
		users:     users,
		databases: databases,
		sessions:  sessions,
	}
}

func (s *groupService) List(ctx context.Context, params pagination.PaginationParams) ([]domain.Group, int64, error) {
	return s.groups.List(ctx, params)
}

func (s *groupService) Get(ctx context.Context, id uint) (domain.Group, error) {
	return s.groups.FindByID(ctx, id)
}

func (s *groupService) Create(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {

	input.Name = strings.TrimSpace(input.Name)

	accesses, err := normalizeAccesses(input.DatabaseAccesses)
	if err != nil {
		return domain.Group{}, err
	}
	if input.Name == "" {
		return domain.Group{}, domain.ErrInvalidInput
	}

	exists, err := s.databases.ExistsAll(ctx, databaseIDs(accesses))
	if err != nil {
		return domain.Group{}, err
	}
	if !exists {
		return domain.Group{}, domain.ErrDatabaseNotFound
	}

	group := domain.Group{
		Name:             input.Name,
		DatabaseAccesses: accesses,
	}

	return s.groups.Create(ctx, group)
}

func (s *groupService) Update(ctx context.Context, id uint, input domain.UpdateGroupInput) (domain.Group, error) {

	current, err := s.groups.FindByID(ctx, id)
	if err != nil {
		return domain.Group{}, err
	}

	accessChanged := false

	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)

		if current.Name == "" {
			return domain.Group{}, domain.ErrInvalidInput
		}
	}

	if input.DatabaseAccesses != nil {
		accesses, err := normalizeAccesses(*input.DatabaseAccesses)
		if err != nil {
			return domain.Group{}, err
		}

		exists, err := s.databases.ExistsAll(ctx, databaseIDs(accesses))
		if err != nil {
			return domain.Group{}, err
		}
		if !exists {
			return domain.Group{}, domain.ErrDatabaseNotFound
		}

		accessChanged = !sameAccesses(current.DatabaseAccesses, accesses)

		current.DatabaseAccesses = accesses
	}

	updated, err := s.groups.Update(ctx, current)
	if err != nil {
		return domain.Group{}, err
	}

	if accessChanged {
		userIDs, err := s.groups.ListUserIDs(ctx, id)
		if err != nil {
			return domain.Group{}, err
		}
		if err := s.revokeUsers(ctx, userIDs); err != nil {
			return domain.Group{}, err
		}
	}

	return updated, nil
}

func (s *groupService) Delete(ctx context.Context, id uint) error {

	if _, err := s.groups.FindByID(ctx, id); err != nil {
		return err
	}

	userIDs, err := s.groups.ListUserIDs(ctx, id)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		user, err := s.users.FindByID(ctx, userID)
		if err != nil {
			return err
		}

		groupCount, err := s.groups.CountUserGroups(ctx, userID)
		if err != nil {
			return err
		}

		if user.Role == "user" && groupCount <= 1 {
			return domain.ErrUserMustHaveGroup
		}
	}

	if err := s.groups.Delete(ctx, id); err != nil {
		return err
	}

	return s.revokeUsers(ctx, userIDs)
}

func (s *groupService) AddUser(ctx context.Context, groupID uint, userID uint) error {

	if _, err := s.groups.FindByID(ctx, groupID); err != nil {
		return err
	}
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return err
	}
	if err := s.groups.AddUser(ctx, groupID, userID); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, userID)
}

func (s *groupService) RemoveUser(ctx context.Context, groupID uint, userID uint) error {
	if _, err := s.groups.FindByID(ctx, groupID); err != nil {
		return err
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	memberIDs, err := s.groups.ListUserIDs(ctx, groupID)
	if err != nil {
		return err
	}
	if !containsUserID(memberIDs, userID) {
		return domain.ErrMembershipNotFound
	}

	groupCount, err := s.groups.CountUserGroups(ctx, userID)
	if err != nil {
		return err
	}

	if user.Role == "user" && groupCount <= 1 {
		return domain.ErrUserMustHaveGroup
	}

	if err := s.groups.RemoveUser(ctx, groupID, userID); err != nil {
		return err
	}

	return s.sessions.RevokeUser(ctx, userID)
}

func (s *groupService) EffectiveDatabaseAccess(ctx context.Context, userID uint, databaseID uint) (domain.AccessType, error) {
	accesses, err := s.groups.ListAccessTypes(ctx, userID, databaseID)
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

func (s *groupService) revokeUsers(ctx context.Context, userIDs []uint) error {
	for _, userID := range userIDs {
		err := s.sessions.RevokeUser(ctx, userID)
		if err != nil {
			return err
		}
	}

	return nil
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

	sort.Slice(result, func(i int, j int) bool {
		return result[i].DatabaseID < result[j].DatabaseID
	})

	return result, nil
}

func databaseIDs(accesses []domain.DatabaseAccess) []uint {
	ids := make([]uint, 0, len(accesses))
	for _, access := range accesses {
		ids = append(ids, access.DatabaseID)
	}

	return ids
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

func containsUserID(userIDs []uint, target uint) bool {
	for _, userID := range userIDs {
		if userID == target {
			return true
		}
	}

	return false
}
