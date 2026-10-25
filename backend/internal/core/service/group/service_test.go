package group_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestListDelegatesToRepository(t *testing.T) {
	params := pagination.PaginationParams{Page: 2, Limit: 5}
	want := []domain.Group{{ID: 1, Name: "team"}}
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			ListFunc: func(_ context.Context, got pagination.PaginationParams) ([]domain.Group, int64, error) {
				if got != params {
					t.Fatalf("params = %+v, want %+v", got, params)
				}
				return want, 7, nil
			},
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	groups, total, err := service.List(context.Background(), params)
	if err != nil || !reflect.DeepEqual(groups, want) || total != 7 {
		t.Fatalf("List() = %#v, %d, %v", groups, total, err)
	}
}

func TestGetDelegatesToRepository(t *testing.T) {
	want := domain.Group{ID: 4, Name: "team"}
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
				if id != 4 {
					t.Fatalf("id = %d, want 4", id)
				}
				return want, nil
			},
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	group, err := service.Get(context.Background(), 4)
	if err != nil || !reflect.DeepEqual(group, want) {
		t.Fatalf("Get() = %+v, %v", group, err)
	}
}

func TestCreateNormalizesAndSortsDatabaseAccesses(t *testing.T) {
	databases := &testutil.DatabaseRepository{
		ExistsAllFunc: func(
			_ context.Context,
			ids []uint,
		) (bool, error) {
			if !reflect.DeepEqual(
				ids,
				[]uint{1, 2},
			) {
				t.Fatalf(
					"database IDs = %v, want [1 2]",
					ids,
				)
			}

			return true, nil
		},
	}

	var captured domain.Group

	groups := &testutil.GroupRepository{
		CreateFunc: func(
			_ context.Context,
			group domain.Group,
		) (domain.Group, error) {
			captured = group
			group.ID = 10

			return group, nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		&testutil.UserRepository{},
		databases,
		&testutil.SessionService{},
	)

	created, err := service.Create(
		context.Background(),
		domain.CreateGroupInput{
			Name: "  analyst  ",
			DatabaseAccesses: []domain.DatabaseAccess{
				{
					DatabaseID: 2,
					AccessType: domain.AccessViewEdit,
				},
				{
					DatabaseID: 1,
					AccessType: domain.AccessViewOnly,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != 10 ||
		captured.Name != "analyst" {
		t.Fatalf(
			"unexpected group: %+v",
			captured,
		)
	}

	if captured.DatabaseAccesses[0].DatabaseID != 1 ||
		captured.DatabaseAccesses[1].DatabaseID != 2 {
		t.Fatalf(
			"accesses were not sorted: %+v",
			captured.DatabaseAccesses,
		)
	}
}

func TestCreateRequiresAtLeastOneDatabase(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	_, err := service.Create(
		context.Background(),
		domain.CreateGroupInput{
			Name: "empty",
		},
	)

	if !errors.Is(
		err,
		domain.ErrGroupMustHaveDatabase,
	) {
		t.Fatalf(
			"error = %v, want ErrGroupMustHaveDatabase",
			err,
		)
	}
}

func TestCreateRejectsInvalidDatabaseAccess(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	_, err := service.Create(
		context.Background(),
		domain.CreateGroupInput{
			Name: "analyst",
			DatabaseAccesses: []domain.DatabaseAccess{{
				DatabaseID: 0,
				AccessType: domain.AccessViewOnly,
			}},
		},
	)

	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("error = %v, want ErrInvalidDatabaseAccess", err)
	}
}

func TestCreateRejectsDuplicateDatabaseAccess(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	_, err := service.Create(
		context.Background(),
		domain.CreateGroupInput{
			Name: "analyst",
			DatabaseAccesses: []domain.DatabaseAccess{
				{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				{DatabaseID: 1, AccessType: domain.AccessViewEdit},
			},
		},
	)

	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("error = %v, want ErrInvalidDatabaseAccess", err)
	}
}

func TestEffectiveDatabaseAccessUsesMaximumPrivilege(t *testing.T) {
	groups := &testutil.GroupRepository{
		ListAccessTypesFunc: func(
			context.Context,
			uint,
			uint,
		) ([]domain.AccessType, error) {
			return []domain.AccessType{
				domain.AccessViewOnly,
				domain.AccessViewEdit,
			}, nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	access, err := service.EffectiveDatabaseAccess(
		context.Background(),
		1,
		2,
	)

	if err != nil {
		t.Fatalf(
			"EffectiveDatabaseAccess() error = %v",
			err,
		)
	}

	if access != domain.AccessViewEdit {
		t.Fatalf(
			"access = %q, want %q",
			access,
			domain.AccessViewEdit,
		)
	}
}

func TestEffectiveDatabaseAccessDeniesMissingGrant(t *testing.T) {
	groups := &testutil.GroupRepository{
		ListAccessTypesFunc: func(
			context.Context,
			uint,
			uint,
		) ([]domain.AccessType, error) {
			return nil, nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	_, err := service.EffectiveDatabaseAccess(
		context.Background(),
		1,
		2,
	)

	if !errors.Is(
		err,
		domain.ErrDatabaseAccessDenied,
	) {
		t.Fatalf(
			"error = %v, want ErrDatabaseAccessDenied",
			err,
		)
	}
}

func TestAddUserRevokesSession(t *testing.T) {
	groups := &testutil.GroupRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.Group, error) {
			return domain.Group{ID: 2}, nil
		},

		AddUserFunc: func(
			context.Context,
			uint,
			uint,
		) error {
			return nil
		},
	}

	users := &testutil.UserRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.User, error) {
			return domain.User{ID: 7}, nil
		},
	}

	revoked := false

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			_ context.Context,
			userID uint,
		) error {
			revoked = userID == 7
			return nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		users,
		&testutil.DatabaseRepository{},
		sessions,
	)

	if err := service.AddUser(
		context.Background(),
		2,
		7,
	); err != nil {
		t.Fatalf(
			"AddUser() error = %v",
			err,
		)
	}

	if !revoked {
		t.Fatal(
			"membership change must revoke affected user session",
		)
	}
}

func TestRemoveUserPreventsRegularUserFromLosingLastGroup(t *testing.T) {
	groups := &testutil.GroupRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.Group, error) {
			return domain.Group{ID: 2}, nil
		},

		ListUserIDsFunc: func(
			context.Context,
			uint,
		) ([]uint, error) {
			return []uint{7}, nil
		},

		CountUserGroupsFunc: func(
			context.Context,
			uint,
		) (int64, error) {
			return 1, nil
		},
	}

	// continue existing test
	users := &testutil.UserRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.User, error) {
			return domain.User{
				ID:   7,
				Role: domain.RoleUser,
			}, nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		users,
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	err := service.RemoveUser(context.Background(), 2, 7)
	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("error = %v, want ErrUserMustHaveGroup", err)
	}
}

func TestRemoveUserRevokesSessionAfterRemovingMembership(t *testing.T) {
	removed := false
	revoked := false
	groups := &testutil.GroupRepository{
		FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
			return domain.Group{ID: 2}, nil
		},
		ListUserIDsFunc: func(context.Context, uint) ([]uint, error) {
			return []uint{7}, nil
		},
		CountUserGroupsFunc: func(context.Context, uint) (int64, error) {
			return 2, nil
		},
		RemoveUserFunc: func(context.Context, uint, uint) error {
			removed = true
			return nil
		},
	}
	users := &testutil.UserRepository{
		FindByIDFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{ID: 7, Role: domain.RoleUser}, nil
		},
	}
	sessions := &testutil.SessionService{
		RevokeUserFunc: func(context.Context, uint) error {
			revoked = true
			return nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		users,
		&testutil.DatabaseRepository{},
		sessions,
	)

	if err := service.RemoveUser(context.Background(), 2, 7); err != nil {
		t.Fatalf("RemoveUser() error = %v", err)
	}
	if !removed || !revoked {
		t.Fatalf("removed=%v revoked=%v, want both true", removed, revoked)
	}
}

func TestRemoveUserRejectsNonMember(t *testing.T) {
	groups := &testutil.GroupRepository{
		FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
			return domain.Group{ID: 2}, nil
		},
		ListUserIDsFunc: func(context.Context, uint) ([]uint, error) {
			return []uint{8}, nil
		},
	}
	users := &testutil.UserRepository{
		FindByIDFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{ID: 7, Role: domain.RoleUser}, nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		users,
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	err := service.RemoveUser(context.Background(), 2, 7)
	if !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatalf("error = %v, want ErrMembershipNotFound", err)
	}
}

func TestUpdateAccessRevokesAllGroupMembers(t *testing.T) {
	current := domain.Group{
		ID:   2,
		Name: "team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{
				DatabaseID: 1,
				AccessType: domain.AccessViewOnly,
			},
		},
	}

	groups := &testutil.GroupRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.Group, error) {
			return current, nil
		},

		UpdateFunc: func(
			_ context.Context,
			group domain.Group,
		) (domain.Group, error) {
			return group, nil
		},

		ListUserIDsFunc: func(
			context.Context,
			uint,
		) ([]uint, error) {
			return []uint{3, 4}, nil
		},
	}

	databases := &testutil.DatabaseRepository{
		ExistsAllFunc: func(
			context.Context,
			[]uint,
		) (bool, error) {
			return true, nil
		},
	}

	var revoked []uint

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			_ context.Context,
			id uint,
		) error {
			revoked = append(revoked, id)
			return nil
		},
	}

	service := groupservice.NewGroupService(
		groups,
		&testutil.UserRepository{},
		databases,
		sessions,
	)

	accesses := []domain.DatabaseAccess{
		{
			DatabaseID: 1,
			AccessType: domain.AccessViewEdit,
		},
	}

	_, err := service.Update(
		context.Background(),
		2,
		domain.UpdateGroupInput{
			DatabaseAccesses: &accesses,
		},
	)

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !reflect.DeepEqual(
		revoked,
		[]uint{3, 4},
	) {
		t.Fatalf(
			"revoked = %v, want [3 4]",
			revoked,
		)
	}
}

func TestDeleteRemovesGroupAndRevokesMembers(t *testing.T) {
	deleted := false
	revoked := []uint{}
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
				return domain.Group{ID: 2}, nil
			},
			ListUserIDsFunc: func(context.Context, uint) ([]uint, error) {
				return []uint{7, 8}, nil
			},
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) {
				return 2, nil
			},
			DeleteFunc: func(context.Context, uint) error {
				deleted = true
				return nil
			},
		},
		&testutil.UserRepository{
			FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
				return domain.User{ID: id, Role: domain.RoleUser}, nil
			},
		},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{
			RevokeUserFunc: func(_ context.Context, id uint) error {
				revoked = append(revoked, id)
				return nil
			},
		},
	)

	if err := service.Delete(context.Background(), 2); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted || !reflect.DeepEqual(revoked, []uint{7, 8}) {
		t.Fatalf("deleted=%v revoked=%v", deleted, revoked)
	}
}
