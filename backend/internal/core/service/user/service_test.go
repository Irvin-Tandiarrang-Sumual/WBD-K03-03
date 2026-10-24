package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateHashesPasswordAndCreatesRegularUser(t *testing.T) {
	groups := &testutil.GroupRepository{
		ExistsAllFunc: func(
			_ context.Context,
			ids []uint,
		) (bool, error) {
			return len(ids) == 2 &&
				ids[0] == 1 &&
				ids[1] == 2, nil
		},
	}

	var captured domain.User

	users := &testutil.UserRepository{
		CreateFunc: func(
			_ context.Context,
			user domain.User,
			groupIDs []uint,
		) (domain.User, error) {
			captured = user
			user.ID = 8

			return user, nil
		},
	}

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			context.Context,
			uint,
		) error {
			return nil
		},
	}

	service := userservice.NewService(
		users,
		groups,
		sessions,
	)

	created, err := service.Create(
		context.Background(),
		domain.CreateUserInput{
			Username: "  alice  ",
			Password: "secret123",
			GroupIDs: []uint{1, 2},
		},
	)

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != 8 ||
		captured.Username != "alice" ||
		captured.Role != domain.RoleUser {
		t.Fatalf(
			"unexpected user: %+v",
			captured,
		)
	}

	if captured.Password == "secret123" {
		t.Fatal("password was stored as plaintext")
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(captured.Password),
		[]byte("secret123"),
	) != nil {
		t.Fatal(
			"stored password is not a bcrypt hash of input",
		)
	}
}

func TestCreateRejectsInvalidGroupIDs(t *testing.T) {
	groups := &testutil.GroupRepository{
		ExistsAllFunc: func(
			context.Context,
			[]uint,
		) (bool, error) {
			return false, nil
		},
	}

	createCalled := false

	users := &testutil.UserRepository{
		CreateFunc: func(
			context.Context,
			domain.User,
			[]uint,
		) (domain.User, error) {
			createCalled = true
			return domain.User{}, nil
		},
	}

	service := userservice.NewService(
		users,
		groups,
		&testutil.SessionService{},
	)

	_, err := service.Create(
		context.Background(),
		domain.CreateUserInput{
			Username: "alice",
			Password: "secret123",
			GroupIDs: []uint{99},
		},
	)

	if !errors.Is(err, domain.ErrInvalidGroupsIDs) {
		t.Fatalf(
			"error = %v, want ErrInvalidGroupsIDs",
			err,
		)
	}

	if createCalled {
		t.Fatal(
			"repository Create() must not run for invalid groups",
		)
	}
}

func TestUpdatePasswordRevokesSessions(t *testing.T) {
	users := &testutil.UserRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.User, error) {
			return domain.User{
				ID:       3,
				Username: "alice",
				Role:     domain.RoleUser,
			}, nil
		},

		UpdateFunc: func(
			_ context.Context,
			id uint,
			username *string,
			passwordHash *string,
		) (domain.User, error) {
			if id != 3 || passwordHash == nil {
				t.Fatal(
					"password hash was not sent to repository",
				)
			}

			if bcrypt.CompareHashAndPassword(
				[]byte(*passwordHash),
				[]byte("new-secret"),
			) != nil {
				t.Fatal(
					"updated password is not hashed correctly",
				)
			}

			return domain.User{
				ID:       3,
				Username: "alice",
				Role:     domain.RoleUser,
			}, nil
		},
	}

	revoked := false

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			_ context.Context,
			userID uint,
		) error {
			revoked = userID == 3
			return nil
		},
	}

	service := userservice.NewService(
		users,
		&testutil.GroupRepository{},
		sessions,
	)

	password := "new-secret"

	_, err := service.Update(
		context.Background(),
		3,
		domain.UpdateUserInput{
			Password: &password,
		},
	)

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !revoked {
		t.Fatal(
			"password change must revoke existing sessions",
		)
	}
}

func TestSetRoleRevokesSessions(t *testing.T) {
	users := &testutil.UserRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.User, error) {
			return domain.User{
				ID:   4,
				Role: domain.RoleUser,
			}, nil
		},

		SetRoleFunc: func(
			_ context.Context,
			id uint,
			role string,
		) (domain.User, error) {
			return domain.User{
				ID:   id,
				Role: role,
			}, nil
		},
	}

	revoked := false

	sessions := &testutil.SessionService{
		RevokeUserFunc: func(
			_ context.Context,
			userID uint,
		) error {
			revoked = userID == 4
			return nil
		},
	}

	service := userservice.NewService(
		users,
		&testutil.GroupRepository{},
		sessions,
	)

	updated, err := service.SetRole(
		context.Background(),
		4,
		domain.RoleAdmin,
	)

	if err != nil {
		t.Fatalf("SetRole() error = %v", err)
	}

	if updated.Role != domain.RoleAdmin || !revoked {
		t.Fatalf(
			"updated=%+v revoked=%v",
			updated,
			revoked,
		)
	}
}

func TestSetRoleRejectsInvalidRole(t *testing.T) {
	service := userservice.NewService(
		&testutil.UserRepository{},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)

	_, err := service.SetRole(
		context.Background(),
		1,
		"superadmin",
	)

	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf(
			"error = %v, want ErrInvalidRole",
			err,
		)
	}
}

func TestSetRoleRejectsRegularUserWithoutGroup(t *testing.T) {
	users := &testutil.UserRepository{
		FindByIDFunc: func(
			context.Context,
			uint,
		) (domain.User, error) {
			return domain.User{
				ID:   5,
				Role: domain.RoleAdmin,
			}, nil
		},
	}

	groups := &testutil.GroupRepository{
		CountUserGroupsFunc: func(
			context.Context,
			uint,
		) (int64, error) {
			return 0, nil
		},
	}

	service := userservice.NewService(
		users,
		groups,
		&testutil.SessionService{},
	)

	_, err := service.SetRole(
		context.Background(),
		5,
		domain.RoleUser,
	)

	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf(
			"error = %v, want ErrUserMustHaveGroup",
			err,
		)
	}
}