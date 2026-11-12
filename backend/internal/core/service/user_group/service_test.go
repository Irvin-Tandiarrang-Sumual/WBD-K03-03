package usergroup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	usergroupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user_group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestAddUser_GroupLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group not found")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(&testutil.UserGroupRepository{}, groupRepo, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.AddUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestAddUser_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user not found")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(&testutil.UserGroupRepository{}, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.AddUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestAddUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to add membership in database")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		AddUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.AddUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestAddUser_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to revoke user session")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		AddUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.AddUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestAddUser_Success_AddsUserMembership(t *testing.T) {
	// Arrange
	capturedGroupID := uint(0)
	capturedUserID := uint(0)
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		AddUserFunc: func(_ context.Context, gID uint, uID uint) error {
			capturedGroupID = gID
			capturedUserID = uID
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.AddUser(context.Background(), 5, 20)

	// Assert
	if err != nil || capturedGroupID != 5 || capturedUserID != 20 {
		t.Fatalf("expected membership (group 5, user 20) added, got group: %d, user: %d, err: %v", capturedGroupID, capturedUserID, err)
	}
}

func TestAddUser_Success_RevokesUserSessions(t *testing.T) {
	// Arrange
	revokedUserID := uint(0)
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		AddUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.AddUser(context.Background(), 5, 20)

	// Assert
	if err != nil || revokedUserID != 20 {
		t.Fatalf("expected sessions for user 20 to be revoked on add, got %d, err: %v", revokedUserID, err)
	}
}

func TestRemoveUser_GroupLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group lookup failed")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(&testutil.UserGroupRepository{}, groupRepo, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user lookup failed")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(&testutil.UserGroupRepository{}, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_FindMembersFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("find group members query failure")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return nil, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_UserNotInGroup_ReturnsErrMembershipNotFound(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{1, 2, 3}, nil // User 10 is NOT in group!
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatalf("expected ErrMembershipNotFound, got %v", err)
	}
}

func TestRemoveUser_CountGroupsFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("count user groups query failure")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 0, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_RoleUserWithSingleGroup_ReturnsErrUserMustHaveGroup(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 1, nil // Only 1 group: cannot remove RoleUser!
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("expected ErrUserMustHaveGroup, got %v", err)
	}
}

func TestRemoveUser_RoleAdminWithSingleGroup_Success(t *testing.T) {
	// Arrange
	removed := false
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleAdmin}, nil // RoleAdmin can have 0 groups
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{99}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 1, nil
		},
		RemoveUserFunc: func(_ context.Context, _ uint, _ uint) error {
			removed = true
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.RemoveUser(context.Background(), 1, 99)

	// Assert
	if err != nil || !removed {
		t.Fatalf("expected RoleAdmin to be successfully removed from group even if groupCount=1, err: %v, removed: %v", err, removed)
	}
}

func TestRemoveUser_RoleUserWithMultipleGroups_Success(t *testing.T) {
	// Arrange
	removed := false
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 2, nil // Has 2 groups, safe to remove from one
		},
		RemoveUserFunc: func(_ context.Context, _ uint, _ uint) error {
			removed = true
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if err != nil || !removed {
		t.Fatalf("expected RoleUser with multiple groups to be removed, err: %v, removed: %v", err, removed)
	}
}

func TestRemoveUser_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("remove user database failure")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 2, nil
		},
		RemoveUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, &testutil.SessionService{})

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("revoke user session failure")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 2, nil
		},
		RemoveUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRemoveUser_Success_RevokesUserSessions(t *testing.T) {
	// Arrange
	revokedUserID := uint(0)
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{ID: id}, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleUser}, nil
		},
	}
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		CountByUserIDFunc: func(_ context.Context, _ uint) (int64, error) {
			return 2, nil
		},
		RemoveUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUserFunc: func(_ context.Context, uID uint) error {
			revokedUserID = uID
			return nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionSvc)

	// Act
	err := service.RemoveUser(context.Background(), 1, 10)

	// Assert
	if err != nil || revokedUserID != 10 {
		t.Fatalf("expected sessions for user 10 to be revoked on remove, got %d, err: %v", revokedUserID, err)
	}
}

func TestAuthorize_AccessLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("access query failure")
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, _ uint, _ uint) ([]domain.AccessType, error) {
			return nil, expectedErr
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.Authorize(context.Background(), 1, 5, domain.AccessViewOnly)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestAuthorize_NoAccessTypesGranted_ReturnsErrDatabaseAccessDenied(t *testing.T) {
	// Arrange
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, _ uint, _ uint) ([]domain.AccessType, error) {
			return []domain.AccessType{}, nil // No groups grant access to this database
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.Authorize(context.Background(), 1, 5, domain.AccessViewOnly)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("expected ErrDatabaseAccessDenied, got %v", err)
	}
}

func TestAuthorize_ViewOnlyGrantedWhenViewEditRequired_ReturnsErrDatabaseAccessDenied(t *testing.T) {
	// Arrange
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, _ uint, _ uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly}, nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.Authorize(context.Background(), 1, 5, domain.AccessViewEdit)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseAccessDenied) {
		t.Fatalf("expected ErrDatabaseAccessDenied when ViewEdit is required but only ViewOnly is granted, got %v", err)
	}
}

func TestAuthorize_ViewOnlyGrantedWhenViewOnlyRequired_ReturnsNil(t *testing.T) {
	// Arrange
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, _ uint, _ uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly}, nil
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.Authorize(context.Background(), 1, 5, domain.AccessViewOnly)

	// Assert
	if err != nil {
		t.Fatalf("expected authorized nil, got %v", err)
	}
}

func TestAuthorize_MultipleGroupsWithOneViewEdit_AuthorizesViewEdit(t *testing.T) {
	// Arrange
	userGroupRepo := &testutil.UserGroupRepository{
		FindAllAccessTypesFunc: func(_ context.Context, _ uint, _ uint) ([]domain.AccessType, error) {
			return []domain.AccessType{domain.AccessViewOnly, domain.AccessViewEdit}, nil // Cumulative
		},
	}
	service := usergroupservice.NewUserGroupService(userGroupRepo, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.SessionService{})

	// Act
	err := service.Authorize(context.Background(), 1, 5, domain.AccessViewEdit)

	// Assert
	if err != nil {
		t.Fatalf("expected authorized nil when at least one group grants ViewEdit, got %v", err)
	}
}
