package user_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"golang.org/x/crypto/bcrypt"
)

func TestListDelegatesToRepository(t *testing.T) {
	params := pagination.PaginationParams{Page: 1, Limit: 10}
	want := []domain.User{{ID: 1, Username: "alice"}}
	service := userservice.NewUserService(
		&testutil.UserRepository{
			ListFunc: func(_ context.Context, got pagination.PaginationParams) ([]domain.User, int64, error) {
				if got != params {
					t.Fatalf("params = %+v, want %+v", got, params)
				}
				return want, 3, nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)

	users, total, err := service.GetAllUsers(context.Background(), params)
	if err != nil || !reflect.DeepEqual(users, want) || total != 3 {
		t.Fatalf("List() = %#v, %d, %v", users, total, err)
	}
}

func TestGetDelegatesToRepository(t *testing.T) {
	want := domain.User{ID: 4, Username: "alice"}
	service := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
				if id != 4 {
					t.Fatalf("id = %d, want 4", id)
				}
				return want, nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)

	user, err := service.GetUserByID(context.Background(), 4)
	if err != nil || !reflect.DeepEqual(user, want) {
		t.Fatalf("Get() = %+v, %v", user, err)
	}
}

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

	service := userservice.NewUserService(
		users,
		groups,
		sessions,
	)

	created, err := service.CreateUser(
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

	service := userservice.NewUserService(
		users,
		groups,
		&testutil.SessionService{},
	)

	_, err := service.CreateUser(
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

	service := userservice.NewUserService(
		users,
		&testutil.GroupRepository{},
		sessions,
	)

	password := "new-secret"

	_, err := service.UpdateUser(
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

func TestUpdateUserRoleRevokesSessions(t *testing.T) {
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

		UpdateUserRoleFunc: func(
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

	service := userservice.NewUserService(
		users,
		&testutil.GroupRepository{},
		sessions,
	)

	updated, err := service.UpdateUserRole(
		context.Background(),
		4,
		domain.RoleAdmin,
	)

	if err != nil {
		t.Fatalf("UpdateUserRole() error = %v", err)
	}

	if updated.Role != domain.RoleAdmin || !revoked {
		t.Fatalf(
			"updated=%+v revoked=%v",
			updated,
			revoked,
		)
	}
}

func TestUpdateUserRoleRejectsInvalidRole(t *testing.T) {
	service := userservice.NewUserService(
		&testutil.UserRepository{},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)

	_, err := service.UpdateUserRole(
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

func TestUpdateUserRoleRejectsRegularUserWithoutGroup(t *testing.T) {
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

	service := userservice.NewUserService(
		users,
		groups,
		&testutil.SessionService{},
	)

	_, err := service.UpdateUserRole(
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

func TestDeleteRemovesUserAndRevokesSessions(t *testing.T) {
	deleted := false
	revoked := false
	service := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 6}, nil
			},
			DeleteFunc: func(context.Context, uint) error {
				deleted = true
				return nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{
			RevokeUserFunc: func(context.Context, uint) error {
				revoked = true
				return nil
			},
		},
	)

	if err := service.DeleteUser(context.Background(), 6); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted || !revoked {
		t.Fatalf("deleted=%v revoked=%v, want both true", deleted, revoked)
	}
}

func TestCreateRejectsInvalidInputs(t *testing.T) {
	tests := []domain.CreateUserInput{
		{Username: " ", Password: "secret", GroupIDs: []uint{1}},
		{Username: "user", Password: "", GroupIDs: []uint{1}},
		{Username: "user", Password: "secret"},
		{Username: "user", Password: string(make([]byte, 73)), GroupIDs: []uint{1}},
	}
	for index, input := range tests {
		service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{})
		_, err := service.CreateUser(context.Background(), input)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("case %d error=%v", index, err)
		}
	}
}

func TestCreatePropagatesGroupAndRepositoryErrors(t *testing.T) {
	groupErr := errors.New("group lookup")
	service := userservice.NewUserService(
		&testutil.UserRepository{},
		&testutil.GroupRepository{ExistsAllFunc: func(context.Context, []uint) (bool, error) { return false, groupErr }},
		&testutil.SessionService{},
	)
	_, err := service.CreateUser(context.Background(), domain.CreateUserInput{Username: "user", Password: "secret", GroupIDs: []uint{1}})
	if !errors.Is(err, groupErr) {
		t.Fatalf("error=%v", err)
	}

	repositoryErr := domain.ErrUsernameTaken
	service = userservice.NewUserService(
		&testutil.UserRepository{CreateFunc: func(context.Context, domain.User, []uint) (domain.User, error) { return domain.User{}, repositoryErr }},
		&testutil.GroupRepository{ExistsAllFunc: func(context.Context, []uint) (bool, error) { return true, nil }},
		&testutil.SessionService{},
	)
	_, err = service.CreateUser(context.Background(), domain.CreateUserInput{Username: "user", Password: "secret", GroupIDs: []uint{1}})
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestUpdateValidationAndErrors(t *testing.T) {
	findErr := domain.ErrUserNotFound
	service := userservice.NewUserService(
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, findErr }},
		&testutil.GroupRepository{}, &testutil.SessionService{},
	)
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{})
	if !errors.Is(err, findErr) {
		t.Fatalf("error=%v", err)
	}

	blank := " "
	emptyPassword := ""
	tooLong := string(make([]byte, 73))
	for _, input := range []domain.UpdateUserInput{{Username: &blank}, {Password: &emptyPassword}, {Password: &tooLong}} {
		service = userservice.NewUserService(
			&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 1}, nil }},
			&testutil.GroupRepository{}, &testutil.SessionService{},
		)
		_, err = service.UpdateUser(context.Background(), 1, input)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("input=%+v error=%v", input, err)
		}
	}
}

func TestUpdateWithoutPasswordDoesNotRevokeSession(t *testing.T) {
	name := " renamed "
	revoked := false
	service := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 1}, nil },
			UpdateFunc: func(_ context.Context, _ uint, username *string, passwordHash *string) (domain.User, error) {
				if username == nil || *username != "renamed" || passwordHash != nil {
					t.Fatalf("username=%v password=%v", username, passwordHash)
				}
				return domain.User{ID: 1, Username: *username}, nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { revoked = true; return nil }},
	)
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Username: &name})
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Fatal("username-only update must not revoke sessions")
	}
}

func TestUpdatePropagatesRepositoryAndRevokeErrors(t *testing.T) {
	updateErr := errors.New("update failed")
	password := "new-secret"
	service := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 1}, nil },
			UpdateFunc:   func(context.Context, uint, *string, *string) (domain.User, error) { return domain.User{}, updateErr },
		},
		&testutil.GroupRepository{}, &testutil.SessionService{},
	)
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Password: &password})
	if !errors.Is(err, updateErr) {
		t.Fatalf("error=%v", err)
	}

	revokeErr := errors.New("revoke failed")
	service = userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 1}, nil },
			UpdateFunc:   func(context.Context, uint, *string, *string) (domain.User, error) { return domain.User{ID: 1}, nil },
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { return revokeErr }},
	)
	_, err = service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Password: &password})
	if !errors.Is(err, revokeErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestUpdateUserRoleNoopDoesNotRevoke(t *testing.T) {
	revoked := false
	service := userservice.NewUserService(
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) {
			return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
		}},
		&testutil.GroupRepository{},
		&testutil.SessionService{RevokeUserFunc: func(context.Context, uint) error { revoked = true; return nil }},
	)
	user, err := service.UpdateUserRole(context.Background(), 1, " ADMIN ")
	if err != nil || user.Role != domain.RoleAdmin {
		t.Fatalf("user=%+v err=%v", user, err)
	}
	if revoked {
		t.Fatal("unchanged role must not revoke sessions")
	}
}

func TestDeletePropagatesDependencyErrors(t *testing.T) {
	findErr := domain.ErrUserNotFound
	service := userservice.NewUserService(
		&testutil.UserRepository{FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{}, findErr }},
		&testutil.GroupRepository{}, &testutil.SessionService{},
	)
	if err := service.DeleteUser(context.Background(), 1); !errors.Is(err, findErr) {
		t.Fatalf("error=%v", err)
	}

	deleteErr := errors.New("delete failed")
	service = userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) { return domain.User{ID: 1}, nil },
			DeleteFunc:   func(context.Context, uint) error { return deleteErr },
		},
		&testutil.GroupRepository{}, &testutil.SessionService{},
	)
	if err := service.DeleteUser(context.Background(), 1); !errors.Is(err, deleteErr) {
		t.Fatalf("error=%v", err)
	}
}

func TestUpdateUserRoleRejectsDemotingLastAdmin(t *testing.T) {
	service := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
			},
			CountByRoleFunc: func(_ context.Context, role string) (int64, error) {
				if role != domain.RoleAdmin {
					t.Fatalf("role = %q, want %q", role, domain.RoleAdmin)
				}
				return 1, nil
			},
		},
		&testutil.GroupRepository{
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) {
				return 1, nil
			},
		},
		&testutil.SessionService{},
	)

	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Fatalf("error = %v, want ErrLastAdmin", err)
	}
}

func TestDeleteRejectsLastAdminAndHandlesCountErrors(t *testing.T) {
	// Rejects deleting the last admin
	serviceLastAdmin := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
			},
			CountByRoleFunc: func(context.Context, string) (int64, error) {
				return 1, nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)
	err := serviceLastAdmin.DeleteUser(context.Background(), 1)
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Fatalf("error=%v, want ErrLastAdmin", err)
	}

	// CountByRole error during admin deletion
	countErr := errors.New("count query failed")
	serviceCountErr := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
			},
			CountByRoleFunc: func(context.Context, string) (int64, error) {
				return 0, countErr
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{},
	)
	if err := serviceCountErr.DeleteUser(context.Background(), 1); !errors.Is(err, countErr) {
		t.Fatalf("error=%v, want countErr", err)
	}

	// RevokeUser error during deletion
	revokeErr := errors.New("session revoke failed")
	serviceRevokeErr := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 2, Role: domain.RoleUser}, nil
			},
			DeleteFunc: func(context.Context, uint) error {
				return nil
			},
		},
		&testutil.GroupRepository{},
		&testutil.SessionService{
			RevokeUserFunc: func(context.Context, uint) error {
				return revokeErr
			},
		},
	)
	if err := serviceRevokeErr.DeleteUser(context.Background(), 2); !errors.Is(err, revokeErr) {
		t.Fatalf("error=%v, want revokeErr", err)
	}
}

func TestUpdateUserRolePropagatesCountAndAdminErrors(t *testing.T) {
	// CountUserGroups error when demoting to user
	groupCountErr := errors.New("group count error")
	serviceGroupErr := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
			},
		},
		&testutil.GroupRepository{
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) {
				return 0, groupCountErr
			},
		},
		&testutil.SessionService{},
	)
	_, err := serviceGroupErr.UpdateUserRole(context.Background(), 1, domain.RoleUser)
	if !errors.Is(err, groupCountErr) {
		t.Fatalf("error=%v, want groupCountErr", err)
	}

	// CountByRole error when demoting admin
	adminCountErr := errors.New("admin count error")
	serviceAdminErr := userservice.NewUserService(
		&testutil.UserRepository{
			FindByIDFunc: func(context.Context, uint) (domain.User, error) {
				return domain.User{ID: 1, Role: domain.RoleAdmin}, nil
			},
			CountByRoleFunc: func(context.Context, string) (int64, error) {
				return 0, adminCountErr
			},
		},
		&testutil.GroupRepository{
			CountUserGroupsFunc: func(context.Context, uint) (int64, error) {
				return 2, nil
			},
		},
		&testutil.SessionService{},
	)
	_, err = serviceAdminErr.UpdateUserRole(context.Background(), 1, domain.RoleUser)
	if !errors.Is(err, adminCountErr) {
		t.Fatalf("error=%v, want adminCountErr", err)
	}
}
