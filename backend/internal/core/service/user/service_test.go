package user_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
	"golang.org/x/crypto/bcrypt"
)

func TestGetAllUsers_ValidParams_ReturnsUsers(t *testing.T) {
	// Arrange
	expectedUsers := []domain.User{
		{ID: 1, Username: "alice", Role: domain.RoleAdmin},
		{ID: 2, Username: "bob", Role: domain.RoleUser},
	}
	userRepo := &testutil.UserRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.User, int64, error) {
			return expectedUsers, 2, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	users, _, _ := service.GetAllUsers(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !reflect.DeepEqual(users, expectedUsers) {
		t.Fatalf("expected users %+v, got %+v", expectedUsers, users)
	}
}

func TestGetAllUsers_ValidParams_ReturnsTotal(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.User, int64, error) {
			return []domain.User{{ID: 1, Username: "alice"}}, 42, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, total, _ := service.GetAllUsers(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if total != 42 {
		t.Fatalf("expected total 42, got %d", total)
	}
}

func TestGetAllUsers_DataFetchFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	userRepo := &testutil.UserRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.User, int64, error) {
			return nil, 0, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, _, err := service.GetAllUsers(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestGetUserByID_ExistingUser_ReturnsUser(t *testing.T) {
	// Arrange
	expectedUser := domain.User{ID: 5, Username: "carol", Role: domain.RoleUser}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return expectedUser, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	user, _ := service.GetUserByID(context.Background(), 5)

	// Assert
	if !reflect.DeepEqual(user, expectedUser) {
		t.Fatalf("expected user %+v, got %+v", expectedUser, user)
	}
}

func TestGetUserByID_DataFetchFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user not found")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, err := service.GetUserByID(context.Background(), 99)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateUser_EmptyUsername_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "",
		Password: "validpassword123",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateUser_WhitespaceOnlyUsername_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "   \t  \n ",
		Password: "validpassword123",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateUser_UsernameExceedsMaxLength_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: strings.Repeat("u", domain.MaxUsernameLength+1),
		Password: "validpassword123",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateUser_EmptyGroupIDs_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: "validpassword123",
		GroupIDs: []uint{},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateUser_EmptyPassword_ReturnsErrPasswordEmpty(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: "",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrPasswordEmpty) {
		t.Fatalf("expected ErrPasswordEmpty, got %v", err)
	}
}

func TestCreateUser_PasswordExceeds72Bytes_ReturnsErrPasswordTooLong(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: strings.Repeat("A", domain.MaxPasswordBytes+1),
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrPasswordTooLong) {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}

func TestCreateUser_GroupVerificationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group repository check error")
	groupRepo := &testutil.GroupRepository{
		AllExistFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, expectedErr
		},
	}
	service := userservice.NewUserService(&testutil.UserRepository{}, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: "validpassword123",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateUser_GroupNotFound_ReturnsErrInvalidGroupsIDs(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		AllExistFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, nil // Groups do not exist
		},
	}
	service := userservice.NewUserService(&testutil.UserRepository{}, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: "validpassword123",
		GroupIDs: []uint{999},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidGroupsIDs) {
		t.Fatalf("expected ErrInvalidGroupsIDs, got %v", err)
	}
}

func TestCreateUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user save error")
	groupRepo := &testutil.GroupRepository{
		AllExistFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		SaveFunc: func(_ context.Context, _ domain.User, _ []uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "newuser",
		Password: "validpassword123",
		GroupIDs: []uint{1},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateUser_ValidInput_HashesPasswordAndAssignsRoleUser(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		AllExistFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	var capturedSavedUser domain.User
	var capturedGroupIDs []uint
	userRepo := &testutil.UserRepository{
		SaveFunc: func(_ context.Context, u domain.User, gIDs []uint) (domain.User, error) {
			capturedSavedUser = u
			capturedGroupIDs = gIDs
			u.ID = 10
			return u, nil
		},
	}
	service := userservice.NewUserService(userRepo, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "  dave  ",
		Password: "secretPassword123",
		GroupIDs: []uint{1, 2},
	}

	// Act
	_, err := service.CreateUser(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSavedUser.Username != "dave" {
		t.Fatalf("expected trimmed username 'dave', got '%s'", capturedSavedUser.Username)
	}
	if capturedSavedUser.Role != domain.RoleUser {
		t.Fatalf("expected role 'user', got '%s'", capturedSavedUser.Role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(capturedSavedUser.Password), []byte("secretPassword123")); err != nil {
		t.Fatalf("expected valid bcrypt hash for password, err: %v", err)
	}
	if !reflect.DeepEqual(capturedGroupIDs, []uint{1, 2}) {
		t.Fatalf("expected group IDs [1 2], got %v", capturedGroupIDs)
	}
}

func TestCreateUser_ValidInput_ReturnsCreatedUser(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		AllExistFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	expectedUser := domain.User{
		ID:       10,
		Username: "dave",
		Role:     domain.RoleUser,
	}
	userRepo := &testutil.UserRepository{
		SaveFunc: func(_ context.Context, _ domain.User, _ []uint) (domain.User, error) {
			return expectedUser, nil
		},
	}
	service := userservice.NewUserService(userRepo, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	input := domain.CreateUserInput{
		Username: "dave",
		Password: "secretPassword123",
		GroupIDs: []uint{1},
	}

	// Act
	created, err := service.CreateUser(context.Background(), input)

	// Assert
	if err != nil || !reflect.DeepEqual(created, expectedUser) {
		t.Fatalf("expected created user %+v, got %+v, err: %v", expectedUser, created, err)
	}
}

func TestUpdateUser_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user not found")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUser_WhitespaceOnlyUsername_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 1, Username: "old"}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	whitespaceName := "   \t  "

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Username: &whitespaceName})

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateUser_UsernameExceedsMaxLength_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 1, Username: "old"}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	tooLongUsername := strings.Repeat("u", domain.MaxUsernameLength+1)

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Username: &tooLongUsername})

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateUser_EmptyPassword_ReturnsErrPasswordEmpty(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 1, Username: "old"}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	emptyPassword := ""

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Password: &emptyPassword})

	// Assert
	if !errors.Is(err, domain.ErrPasswordEmpty) {
		t.Fatalf("expected ErrPasswordEmpty, got %v", err)
	}
}

func TestUpdateUser_PasswordExceeds72Bytes_ReturnsErrPasswordTooLong(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 1, Username: "old"}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	tooLongPassword := strings.Repeat("A", domain.MaxPasswordBytes+1)

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Password: &tooLongPassword})

	// Assert
	if !errors.Is(err, domain.ErrPasswordTooLong) {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}

func TestUpdateUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("update user repository failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 1, Username: "old"}, nil
		},
		UpdateFunc: func(_ context.Context, _ uint, _ *string, _ *string) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})
	newName := "newname"

	// Act
	_, err := service.UpdateUser(context.Background(), 1, domain.UpdateUserInput{Username: &newName})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUser_PasswordUpdated_RevokesUserSessions(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 7, Username: "old"}, nil
		},
		UpdateFunc: func(_ context.Context, _ uint, _ *string, _ *string) (domain.User, error) {
			return domain.User{ID: 7, Username: "old"}, nil
		},
	}
	revokedUserID := uint(0)
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})
	newPassword := "brandNewPassword123"

	// Act
	_, err := service.UpdateUser(context.Background(), 7, domain.UpdateUserInput{Password: &newPassword})

	// Assert
	if err != nil || revokedUserID != 7 {
		t.Fatalf("expected sessions for user 7 to be revoked on password change, got revokedUserID: %d, err: %v", revokedUserID, err)
	}
}

func TestUpdateUser_OnlyUsernameUpdated_DoesNotRevokeUserSessions(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 7, Username: "old"}, nil
		},
		UpdateFunc: func(_ context.Context, _ uint, _ *string, _ *string) (domain.User, error) {
			return domain.User{ID: 7, Username: "newname"}, nil
		},
	}
	sessionRevoked := false
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			sessionRevoked = true
			return nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})
	newName := "newname"

	// Act
	_, err := service.UpdateUser(context.Background(), 7, domain.UpdateUserInput{Username: &newName})

	// Assert
	if err != nil || sessionRevoked {
		t.Fatalf("expected sessions NOT to be revoked when only username changes, got sessionRevoked: %v, err: %v", sessionRevoked, err)
	}
}

func TestUpdateUser_PasswordRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to revoke sessions")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{ID: 7, Username: "old"}, nil
		},
		UpdateFunc: func(_ context.Context, _ uint, _ *string, _ *string) (domain.User, error) {
			return domain.User{ID: 7, Username: "old"}, nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})
	newPassword := "brandNewPassword123"

	// Act
	_, err := service.UpdateUser(context.Background(), 7, domain.UpdateUserInput{Password: &newPassword})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user not found")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 99)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_CountByLeaderIDFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("count leader database error")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		CountByLeaderIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 0, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 5)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_UserIsGroupLeader_ReturnsErrUserIsGroupLeader(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		CountByLeaderIDFunc: func(_ context.Context, leaderID uint) (int64, error) {
			if leaderID == 5 {
				return 2, nil // leads 2 groups
			}
			return 0, nil
		},
	}
	service := userservice.NewUserService(userRepo, groupRepo, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 5)

	// Assert
	if !errors.Is(err, domain.ErrUserIsGroupLeader) {
		t.Fatalf("expected ErrUserIsGroupLeader, got %v", err)
	}
}

func TestDeleteUser_LastAdminUser_ReturnsErrLastAdmin(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
		CountByRoleFunc: func(_ context.Context, role string) (int64, error) {
			if role == domain.RoleAdmin {
				return 1, nil // Only 1 admin left
			}
			return 0, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 1)

	// Assert
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestDeleteUser_CountAdminFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("count admin database error")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
		CountByRoleFunc: func(_ context.Context, _ string) (int64, error) {
			return 0, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("delete user failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		DeleteFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 2)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("revoke user session failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		DeleteFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 2)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteUser_Success_DeletesUserAndRevokesSessions(t *testing.T) {
	// Arrange
	deletedUserID := uint(0)
	revokedUserID := uint(0)
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		DeleteFunc: func(_ context.Context, id uint) error {
			deletedUserID = id
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})

	// Act
	err := service.DeleteUser(context.Background(), 15)

	// Assert
	if err != nil || deletedUserID != 15 || revokedUserID != 15 {
		t.Fatalf("expected user 15 to be deleted and sessions revoked, got deleted: %d, revoked: %d, err: %v", deletedUserID, revokedUserID, err)
	}
}

func TestUpdateUserRole_InvalidRoleString_ReturnsErrInvalidRole(t *testing.T) {
	// Arrange
	service := userservice.NewUserService(&testutil.UserRepository{}, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, "superadmin_invalid")

	// Assert
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestUpdateUserRole_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user not found")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, err := service.UpdateUserRole(context.Background(), 99, domain.RoleAdmin)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUserRole_SameRole_ReturnsCurrentWithoutCallingRepository(t *testing.T) {
	// Arrange
	currentUser := domain.User{ID: 1, Role: domain.RoleAdmin}
	updateCalled := false
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return currentUser, nil
		},
		UpdateRoleFunc: func(_ context.Context, _ uint, _ string) (domain.User, error) {
			updateCalled = true
			return domain.User{}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	result, err := service.UpdateUserRole(context.Background(), 1, "ADMIN")

	// Assert
	if err != nil || !reflect.DeepEqual(result, currentUser) || updateCalled {
		t.Fatalf("expected idempotent return without calling update, got updateCalled: %v, result: %+v, err: %v", updateCalled, result, err)
	}
}

func TestUpdateUserRole_NilUserGroupRepository_ReturnsErrUserMustHaveGroup(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, nil)

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)

	// Assert
	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("expected ErrUserMustHaveGroup, got %v", err)
	}
}

func TestUpdateUserRole_DemoteToUserWithZeroGroups_ReturnsErrUserMustHaveGroup(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 0, nil // Admin has 0 groups, cannot be demoted to user!
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, userGroupRepo)

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)

	// Assert
	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("expected ErrUserMustHaveGroup, got %v", err)
	}
}

func TestUpdateUserRole_DemoteToUserCountGroupsFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("count user groups error")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 0, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, userGroupRepo)

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUserRole_DemoteLastAdmin_ReturnsErrLastAdmin(t *testing.T) {
	// Arrange
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
		CountByRoleFunc: func(_ context.Context, role string) (int64, error) {
			if role == domain.RoleAdmin {
				return 1, nil // Only 1 admin left!
			}
			return 0, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 1, nil // Has 1 group, but is the last admin
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, userGroupRepo)

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)

	// Assert
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestUpdateUserRole_DemoteAdminCountFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("count admin failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
		CountByRoleFunc: func(_ context.Context, _ string) (int64, error) {
			return 0, expectedErr
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 1, nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, userGroupRepo)

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleUser)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUserRole_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("update role failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		UpdateRoleFunc: func(_ context.Context, _ uint, _ string) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, &testutil.SessionService{}, &testutil.UserGroupRepository{})

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleAdmin)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUserRole_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("session revoke failure")
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		UpdateRoleFunc: func(_ context.Context, id uint, role string) (domain.User, error) {
			return domain.User{ID: id, Role: role}, nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})

	// Act
	_, err := service.UpdateUserRole(context.Background(), 1, domain.RoleAdmin)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateUserRole_PromoteUserToAdmin_SuccessAndRevokesSessions(t *testing.T) {
	// Arrange
	revokedUserID := uint(0)
	expectedUser := domain.User{ID: 10, Role: domain.RoleAdmin}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
		UpdateRoleFunc: func(_ context.Context, id uint, role string) (domain.User, error) {
			return domain.User{ID: id, Role: role}, nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, &testutil.UserGroupRepository{})

	// Act
	updated, err := service.UpdateUserRole(context.Background(), 10, domain.RoleAdmin)

	// Assert
	if err != nil || !reflect.DeepEqual(updated, expectedUser) || revokedUserID != 10 {
		t.Fatalf("expected user 10 to be promoted to admin and sessions revoked, got user: %+v, revokedID: %d, err: %v", updated, revokedUserID, err)
	}
}

func TestUpdateUserRole_DemoteAdminToUserWithGroups_SuccessAndRevokesSessions(t *testing.T) {
	// Arrange
	revokedUserID := uint(0)
	expectedUser := domain.User{ID: 10, Role: domain.RoleUser}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil
		},
		CountByRoleFunc: func(_ context.Context, role string) (int64, error) {
			return 2, nil // 2 admins exist, safe to demote
		},
		UpdateRoleFunc: func(_ context.Context, id uint, role string) (domain.User, error) {
			return domain.User{ID: id, Role: role}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 1, nil // Admin belongs to 1 group, safe to demote
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := userservice.NewUserService(userRepo, &testutil.GroupRepository{}, sessionSvc, userGroupRepo)

	// Act
	updated, err := service.UpdateUserRole(context.Background(), 10, domain.RoleUser)

	// Assert
	if err != nil || !reflect.DeepEqual(updated, expectedUser) || revokedUserID != 10 {
		t.Fatalf("expected user 10 to be demoted to user and sessions revoked, got user: %+v, revokedID: %d, err: %v", updated, revokedUserID, err)
	}
}
