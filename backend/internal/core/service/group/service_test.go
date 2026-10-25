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

func TestUpdateValidationAndDependencyErrors(t *testing.T) {
	findErr := domain.ErrGroupNotFound
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{}, findErr }},
		&testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if _, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{}); !errors.Is(err, findErr) {
		t.Fatalf("find error=%v", err)
	}

	blank := " "
	service = groupservice.NewGroupService(
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1, Name: "old"}, nil }},
		&testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if _, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{Name: &blank}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("blank name error=%v", err)
	}
}

func TestUpdateSameAccessDoesNotRevokeUsers(t *testing.T) {
	accesses := []domain.DatabaseAccess{{DatabaseID: 2, AccessType: domain.AccessViewOnly}}
	listUsersCalled := false
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
				return domain.Group{ID: 1, Name: "team", DatabaseAccesses: accesses}, nil
			},
			UpdateFunc: func(_ context.Context, group domain.Group) (domain.Group, error) { return group, nil },
			ListUserIDsFunc: func(context.Context, uint) ([]uint, error) {
				listUsersCalled = true
				return []uint{3}, nil
			},
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{ExistsAllFunc: func(context.Context, []uint) (bool, error) { return true, nil }},
		&testutil.SessionService{},
	)
	if _, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{DatabaseAccesses: &accesses}); err != nil {
		t.Fatal(err)
	}
	if listUsersCalled {
		t.Fatal("unchanged database access must not revoke sessions")
	}
}

func TestUpdateRejectsMissingDatabaseAndPropagatesRevokeFailure(t *testing.T) {
	newAccesses := []domain.DatabaseAccess{{DatabaseID: 5, AccessType: domain.AccessViewEdit}}
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "team", DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: 2, AccessType: domain.AccessViewOnly}}}, nil
		}},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{ExistsAllFunc: func(context.Context, []uint) (bool, error) { return false, nil }},
		&testutil.SessionService{},
	)
	if _, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{DatabaseAccesses: &newAccesses}); !errors.Is(err, domain.ErrDatabaseNotFound) {
		t.Fatalf("error=%v", err)
	}

	revokeErr := errors.New("revoke failed")
	service = groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
				return domain.Group{ID: 1, Name: "team", DatabaseAccesses: []domain.DatabaseAccess{{DatabaseID: 2, AccessType: domain.AccessViewOnly}}}, nil
			},
			UpdateFunc:      func(_ context.Context, group domain.Group) (domain.Group, error) { return group, nil },
			ListUserIDsFunc: func(context.Context, uint) ([]uint, error) { return []uint{3}, nil },
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{ExistsAllFunc: func(context.Context, []uint) (bool, error) { return true, nil }},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { return revokeErr }},
	)
	if _, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{DatabaseAccesses: &newAccesses}); !errors.Is(err, revokeErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestAddUserPropagatesValidationAndSessionErrors(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{}, domain.ErrGroupNotFound }},
		&testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("error=%v", err)
	}

	service = groupservice.NewGroupService(
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil }},
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, domain.ErrUserNotFound }},
		&testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("error=%v", err)
	}

	revokeErr := errors.New("revoke failed")
	service = groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
			AddUserFunc:  func(context.Context, uint, uint) error { return nil },
		},
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 2}, nil }},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { return revokeErr }},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, revokeErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestEffectiveDatabaseAccessReturnsViewOnlyAndRepositoryError(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{ListAccessTypesFunc: func(context.Context, uint, uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly}, nil
		}},
		&testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	access, err := service.EffectiveDatabaseAccess(context.Background(), 1, 2)
	if err != nil || access != domain.AccessViewOnly {
		t.Fatalf("access=%q err=%v", access, err)
	}

	wantErr := errors.New("query failed")
	service = groupservice.NewGroupService(
		&testutil.GroupRepository{ListAccessTypesFunc: func(context.Context, uint, uint) ([]domain.AccessType, error) { return nil, wantErr }},
		&testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if _, err := service.EffectiveDatabaseAccess(context.Background(), 1, 2); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestDeleteRejectsLastGroupForRegularUser(t *testing.T) {
	deleted := false
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc:        func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
			ListUserIDsFunc:     func(context.Context, uint) ([]uint, error) { return []uint{7}, nil },
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) { return 1, nil },
			DeleteFunc: func(context.Context, uint) error {
				deleted = true
				return nil
			},
		},
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{ID: 7, Role: domain.RoleUser}, nil
		}},
		&testutil.DatabaseRepository{}, &testutil.SessionService{},
	)
	if err := service.Delete(context.Background(), 1); !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("error=%v", err)
	}
	if deleted {
		t.Fatal("group must not be deleted when it is a regular user's last group")
	}
}

func TestCreateGroupHandlesDatabaseExistsErrorAndDuplicateAccess(t *testing.T) {
	// Database ExistsAll error
	dbErr := errors.New("database repo error")
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{
			ExistsAllFunc: func(context.Context, []uint) (bool, error) {
				return false, dbErr
			},
		},
		&testutil.SessionService{},
	)
	_, err := service.Create(context.Background(), domain.CreateGroupInput{
		Name: "dev-team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error=%v, want dbErr", err)
	}

	// Duplicate database ID in access inputs
	_, err = service.Create(context.Background(), domain.CreateGroupInput{
		Name: "dev-team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
			{DatabaseID: 1, AccessType: domain.AccessViewEdit},
		},
	})
	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("error=%v, want ErrInvalidDatabaseAccess", err)
	}
}

func TestUpdateGroupRejectsBlankNameAndHandlesDatabaseExistsError(t *testing.T) {
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
				return domain.Group{ID: 1, Name: "current"}, nil
			},
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	// Blank name
	blank := "   "
	_, err := service.Update(context.Background(), 1, domain.UpdateGroupInput{Name: &blank})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error=%v, want ErrInvalidInput", err)
	}

	// Database ExistsAll error during update
	dbErr := errors.New("db check failed")
	serviceDbErr := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) {
				return domain.Group{ID: 1, Name: "current"}, nil
			},
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{
			ExistsAllFunc: func(context.Context, []uint) (bool, error) {
				return false, dbErr
			},
		},
		&testutil.SessionService{},
	)
	accesses := []domain.DatabaseAccess{{DatabaseID: 2, AccessType: domain.AccessViewOnly}}
	_, err = serviceDbErr.Update(context.Background(), 1, domain.UpdateGroupInput{DatabaseAccesses: &accesses})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error=%v, want dbErr", err)
	}
}

func TestDeleteGroupHandlesDependencyErrors(t *testing.T) {
	// ListUserIDs error
	listErr := errors.New("list users error")
	serviceListErr := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc:    func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
			ListUserIDsFunc: func(context.Context, uint) ([]uint, error) { return nil, listErr },
		},
		&testutil.UserRepository{},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)
	if err := serviceListErr.Delete(context.Background(), 1); !errors.Is(err, listErr) {
		t.Fatalf("error=%v, want listErr", err)
	}

	// User FindByID error inside loop
	userErr := errors.New("user find error")
	serviceUserErr := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc:    func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
			ListUserIDsFunc: func(context.Context, uint) ([]uint, error) { return []uint{10}, nil },
		},
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, userErr },
		},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)
	if err := serviceUserErr.Delete(context.Background(), 1); !errors.Is(err, userErr) {
		t.Fatalf("error=%v, want userErr", err)
	}

	// CountUserGroups error inside loop
	countErr := errors.New("count groups error")
	serviceCountErr := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc:        func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
			ListUserIDsFunc:     func(context.Context, uint) ([]uint, error) { return []uint{10}, nil },
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) { return 0, countErr },
		},
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 10, Role: "user"}, nil },
		},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)
	if err := serviceCountErr.Delete(context.Background(), 1); !errors.Is(err, countErr) {
		t.Fatalf("error=%v, want countErr", err)
	}
}

func TestAddAndRemoveUserPropagatesDependencyErrors(t *testing.T) {
	userErr := errors.New("user lookup failed")
	service := groupservice.NewGroupService(
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
		},
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, userErr },
		},
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)
	// AddUser user lookup error
	if err := service.AddUser(context.Background(), 1, 5); !errors.Is(err, userErr) {
		t.Fatalf("AddUser error=%v, want userErr", err)
	}
	// RemoveUser user lookup error
	if err := service.RemoveUser(context.Background(), 1, 5); !errors.Is(err, userErr) {
		t.Fatalf("RemoveUser error=%v, want userErr", err)
	}
}
