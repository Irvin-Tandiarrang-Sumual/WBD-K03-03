package group_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

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

	service := groupservice.NewService(
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
	service := groupservice.NewService(
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

	service := groupservice.NewService(
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

	service := groupservice.NewService(
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

	service := groupservice.NewService(
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

	service := groupservice.NewService(
		groups,
		users,
		&testutil.DatabaseRepository{},
		&testutil.SessionService{},
	)

	err := service.RemoveUser(
		context.Background(),
		2,
		7,
	)

	if !errors.Is(
		err,
		domain.ErrUserMustHaveGroup,
	) {
		t.Fatalf(
			"error = %v, want ErrUserMustHaveGroup",
			err,
		)
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

	service := groupservice.NewService(
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