package usergroup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	usergroupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user_group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

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

	service := usergroupservice.NewUserGroupService(
		groups,
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.SessionService{},
	)

	access, err := service.GetEffectiveDatabaseAccess(
		context.Background(),
		1,
		2,
	)

	if err != nil {
		t.Fatalf(
			"GetEffectiveDatabaseAccess() error = %v",
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

	service := usergroupservice.NewUserGroupService(
		groups,
		&testutil.GroupRepository{},
		&testutil.UserRepository{},
		&testutil.SessionService{},
	)

	_, err := service.GetEffectiveDatabaseAccess(
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

	service := usergroupservice.NewUserGroupService(
		groups,
		groups,
		users,
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

	service := usergroupservice.NewUserGroupService(
		groups,
		groups,
		users,
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

	service := usergroupservice.NewUserGroupService(
		groups,
		groups,
		users,
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

	service := usergroupservice.NewUserGroupService(
		groups,
		groups,
		users,
		&testutil.SessionService{},
	)

	err := service.RemoveUser(context.Background(), 2, 7)
	if !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatalf("error = %v, want ErrMembershipNotFound", err)
	}
}

func TestAddUserPropagatesValidationAndSessionErrors(t *testing.T) {
	service := usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{},
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{}, domain.ErrGroupNotFound }},
		&testutil.UserRepository{}, &testutil.SessionService{},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("error=%v", err)
	}

	service = usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{},
		&testutil.GroupRepository{FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil }},
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, domain.ErrUserNotFound }},
		&testutil.SessionService{},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("error=%v", err)
	}

	revokeErr := errors.New("revoke failed")
	service = usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{
			AddUserFunc: func(context.Context, uint, uint) error { return nil },
		},
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
		},
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 2}, nil }},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { return revokeErr }},
	)
	if err := service.AddUser(context.Background(), 1, 2); !errors.Is(err, revokeErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestEffectiveDatabaseAccessReturnsViewOnlyAndRepositoryError(t *testing.T) {
	service := usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{ListAccessTypesFunc: func(context.Context, uint, uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly}, nil
		}},
		&testutil.GroupRepository{},
		&testutil.UserRepository{}, &testutil.SessionService{},
	)
	access, err := service.GetEffectiveDatabaseAccess(context.Background(), 1, 2)
	if err != nil || access != domain.AccessViewOnly {
		t.Fatalf("access=%q err=%v", access, err)
	}

	wantErr := errors.New("query failed")
	service = usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{ListAccessTypesFunc: func(context.Context, uint, uint) ([]domain.AccessType, error) { return nil, wantErr }},
		&testutil.GroupRepository{},
		&testutil.UserRepository{}, &testutil.SessionService{},
	)
	if _, err := service.GetEffectiveDatabaseAccess(context.Background(), 1, 2); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestAddAndRemoveUserPropagatesDependencyErrors(t *testing.T) {
	userErr := errors.New("user lookup failed")
	service := usergroupservice.NewUserGroupService(
		&testutil.GroupRepository{},
		&testutil.GroupRepository{
			FindByIDFunc: func(context.Context, uint) (domain.Group, error) { return domain.Group{ID: 1}, nil },
		},
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, userErr },
		},
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
