package group_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/pagination"
)

func TestGetAllGroups_ValidParams_ReturnsGroups(t *testing.T) {
	// Arrange
	expectedGroups := []domain.Group{{ID: 1, Name: "Engineering"}}
	repo := &testutil.GroupRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.Group, int64, error) {
			return expectedGroups, 1, nil
		},
	}
	service := groupservice.NewGroupService(repo, repo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	groups, _, _ := service.GetAllGroups(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !reflect.DeepEqual(groups, expectedGroups) {
		t.Fatalf("expected groups %+v, got %+v", expectedGroups, groups)
	}
}

func TestGetAllGroups_ValidParams_ReturnsTotal(t *testing.T) {
	// Arrange
	repo := &testutil.GroupRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.Group, int64, error) {
			return []domain.Group{{ID: 1, Name: "Engineering"}}, 42, nil
		},
	}
	service := groupservice.NewGroupService(repo, repo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	_, total, _ := service.GetAllGroups(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if total != 42 {
		t.Fatalf("expected total 42, got %d", total)
	}
}

func TestGetAllGroups_DataFetchFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	repo := &testutil.GroupRepository{
		FindAllFunc: func(_ context.Context, _ pagination.PaginationParams) ([]domain.Group, int64, error) {
			return nil, 0, expectedErr
		},
	}
	service := groupservice.NewGroupService(repo, repo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	_, _, err := service.GetAllGroups(context.Background(), pagination.PaginationParams{Page: 1, Limit: 10})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestGetGroupByID_ExistingGroup_ReturnsGroup(t *testing.T) {
	// Arrange
	expectedGroup := domain.Group{ID: 7, Name: "Data Science"}
	repo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return expectedGroup, nil
		},
	}
	service := groupservice.NewGroupService(repo, repo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	group, _ := service.GetGroupByID(context.Background(), 7)

	// Assert
	if !reflect.DeepEqual(group, expectedGroup) {
		t.Fatalf("expected group %+v, got %+v", expectedGroup, group)
	}
}

func TestGetGroupByID_DataFetchFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group not found")
	repo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := groupservice.NewGroupService(repo, repo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	_, err := service.GetGroupByID(context.Background(), 99)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateGroup_EmptyName_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateGroup_WhitespaceOnlyName_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "   \t  \n ",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateGroup_NameExceedsMaxLength_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: strings.Repeat("A", domain.MaxGroupNameLength+1),
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateGroup_EmptyDatabaseAccesses_ReturnsErrGroupMustHaveDatabase(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:             "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrGroupMustHaveDatabase) {
		t.Fatalf("expected ErrGroupMustHaveDatabase, got %v", err)
	}
}

func TestCreateGroup_ZeroDatabaseID_ReturnsErrInvalidDatabaseAccess(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 0, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("expected ErrInvalidDatabaseAccess, got %v", err)
	}
}

func TestCreateGroup_InvalidAccessType_ReturnsErrInvalidDatabaseAccess(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessType("admin_full_access")},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("expected ErrInvalidDatabaseAccess, got %v", err)
	}
}

func TestCreateGroup_DuplicateDatabaseID_ReturnsErrInvalidDatabaseAccess(t *testing.T) {
	// Arrange
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
			{DatabaseID: 1, AccessType: domain.AccessViewEdit},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDatabaseAccess) {
		t.Fatalf("expected ErrInvalidDatabaseAccess, got %v", err)
	}
}

func TestCreateGroup_DatabaseVerificationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database repository failure")
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, expectedErr
		},
	}
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateGroup_DatabaseNotFound_ReturnsErrDatabaseNotFound(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, nil
		},
	}
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name: "Backend Team",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 999, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrDatabaseNotFound) {
		t.Fatalf("expected ErrDatabaseNotFound, got %v", err)
	}
}

func TestCreateGroup_AccessesUnsorted_SavesNormalizedSortedAccesses(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	var capturedSavedGroup domain.Group
	groupRepo := &testutil.GroupRepository{
		SaveFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			capturedSavedGroup = g
			return g, nil
		},
		AddUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, userRepo, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "Multi-DB Team",
		LeaderID: 10,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 5, AccessType: domain.AccessViewEdit},
			{DatabaseID: 2, AccessType: domain.AccessViewOnly},
			{DatabaseID: 8, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, _ = service.CreateGroup(context.Background(), input)

	// Assert
	expectedSortedAccesses := []domain.DatabaseAccess{
		{DatabaseID: 2, AccessType: domain.AccessViewOnly},
		{DatabaseID: 5, AccessType: domain.AccessViewEdit},
		{DatabaseID: 8, AccessType: domain.AccessViewOnly},
	}
	if !reflect.DeepEqual(capturedSavedGroup.DatabaseAccesses, expectedSortedAccesses) {
		t.Fatalf("expected accesses sorted by ID %+v, got %+v", expectedSortedAccesses, capturedSavedGroup.DatabaseAccesses)
	}
}

func TestCreateGroup_TrimmedName_SavesTrimmedName(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	var capturedSavedGroup domain.Group
	groupRepo := &testutil.GroupRepository{
		SaveFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			capturedSavedGroup = g
			return g, nil
		},
		AddUserFunc: func(_ context.Context, _ uint, _ uint) error {
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, userRepo, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "  Padded Name   ",
		LeaderID: 10,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, _ = service.CreateGroup(context.Background(), input)

	// Assert
	if capturedSavedGroup.Name != "Padded Name" {
		t.Fatalf("expected trimmed name 'Padded Name', got '%s'", capturedSavedGroup.Name)
	}
}

func TestCreateGroup_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to save group")
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	groupRepo := &testutil.GroupRepository{
		SaveFunc: func(_ context.Context, _ domain.Group) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, userRepo, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "Engineering",
		LeaderID: 10,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateGroup_ValidInput_ReturnsCreatedGroup(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, id uint) (domain.User, error) {
			return domain.User{ID: id}, nil
		},
	}
	expectedGroup := domain.Group{
		ID:       12,
		Name:     "Engineering",
		LeaderID: 10,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}
	addedLeaderToGroup := false
	groupRepo := &testutil.GroupRepository{
		SaveFunc: func(_ context.Context, _ domain.Group) (domain.Group, error) {
			return expectedGroup, nil
		},
		AddUserFunc: func(_ context.Context, groupID uint, userID uint) error {
			if groupID == 12 && userID == 10 {
				addedLeaderToGroup = true
			}
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, userRepo, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "Engineering",
		LeaderID: 10,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	created, _ := service.CreateGroup(context.Background(), input)

	// Assert
	if !reflect.DeepEqual(created, expectedGroup) {
		t.Fatalf("expected group %+v, got %+v", expectedGroup, created)
	}
	if !addedLeaderToGroup {
		t.Fatalf("expected leader to be added to group membership")
	}
}

func TestCreateGroup_ZeroLeaderID_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "Engineering",
		LeaderID: 0,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateGroup_LeaderNotFound_ReturnsErrUserNotFound(t *testing.T) {
	// Arrange
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	userRepo := &testutil.UserRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	service := groupservice.NewGroupService(&testutil.GroupRepository{}, &testutil.GroupRepository{}, userRepo, dbRepo, &testutil.SessionService{})
	input := domain.CreateGroupInput{
		Name:     "Engineering",
		LeaderID: 999,
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}

	// Act
	_, err := service.CreateGroup(context.Background(), input)

	// Assert
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateGroup_GroupLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group not found")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateGroup_EmptyName_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	emptyName := ""

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{Name: &emptyName})

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateGroup_WhitespaceOnlyName_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	whitespaceName := "   \t  "

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{Name: &whitespaceName})

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateGroup_NameExceedsMaxLength_ReturnsErrInvalidInput(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	tooLongName := strings.Repeat("B", domain.MaxGroupNameLength+1)

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{Name: &tooLongName})

	// Assert
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateGroup_InvalidDatabaseAccess_ReturnsError(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	invalidAccesses := []domain.DatabaseAccess{}

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &invalidAccesses,
	})

	// Assert
	if !errors.Is(err, domain.ErrGroupMustHaveDatabase) {
		t.Fatalf("expected ErrGroupMustHaveDatabase, got %v", err)
	}
}

func TestUpdateGroup_DatabaseNotFound_ReturnsErrDatabaseNotFound(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 999, AccessType: domain.AccessViewOnly},
	}

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	if !errors.Is(err, domain.ErrDatabaseNotFound) {
		t.Fatalf("expected ErrDatabaseNotFound, got %v", err)
	}
}

func TestUpdateGroup_DatabaseVerificationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database repo error")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return false, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 1, AccessType: domain.AccessViewOnly},
	}

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateGroup_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("update failed")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Current Name"}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.Group) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	newName := "Updated Name"

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{Name: &newName})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateGroup_AccessChanged_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("user group repo error")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return nil, expectedErr
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, &testutil.SessionService{})
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 1, AccessType: domain.AccessViewEdit}, // changed from ViewOnly to ViewEdit
	}

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateGroup_AccessChanged_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("session service error")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10, 20}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, _ []uint) error {
			return expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, sessionSvc)
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 2, AccessType: domain.AccessViewOnly}, // changed database ID
	}

	// Act
	_, err := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUpdateGroup_AccessChanged_RevokesUserSessions(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10, 20}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	var capturedRevokedUserIDs []uint
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, userIDs []uint) error {
			capturedRevokedUserIDs = userIDs
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, sessionSvc)
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 1, AccessType: domain.AccessViewEdit},
	}

	// Act
	_, _ = service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	expectedUserIDs := []uint{10, 20}
	if !reflect.DeepEqual(capturedRevokedUserIDs, expectedUserIDs) {
		t.Fatalf("expected revoked users %v, got %v", expectedUserIDs, capturedRevokedUserIDs)
	}
}

func TestUpdateGroup_AccessCountChanged_RevokesUserSessions(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10, 20}, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	var capturedRevokedUserIDs []uint
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, userIDs []uint) error {
			capturedRevokedUserIDs = userIDs
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, sessionSvc)
	newAccesses := []domain.DatabaseAccess{
		{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		{DatabaseID: 2, AccessType: domain.AccessViewOnly}, // Count changed from 1 to 2
	}

	// Act
	_, _ = service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &newAccesses,
	})

	// Assert
	expectedUserIDs := []uint{10, 20}
	if !reflect.DeepEqual(capturedRevokedUserIDs, expectedUserIDs) {
		t.Fatalf("expected revoked users %v, got %v", expectedUserIDs, capturedRevokedUserIDs)
	}
}

func TestUpdateGroup_AccessUnchanged_DoesNotRevokeUserSessions(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
	}
	dbRepo := &testutil.DatabaseRepository{
		ExistsAllFunc: func(_ context.Context, _ []uint) (bool, error) {
			return true, nil
		},
	}
	sessionRevoked := false
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, _ []uint) error {
			sessionRevoked = true
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, dbRepo, sessionSvc)
	sameAccessesInput := []domain.DatabaseAccess{
		{DatabaseID: 1, AccessType: domain.AccessViewOnly}, // Identical accesses
	}

	// Act
	_, _ = service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		DatabaseAccesses: &sameAccessesInput,
	})

	// Assert
	if sessionRevoked {
		t.Fatal("expected sessions NOT to be revoked when database accesses are identical")
	}
}

func TestUpdateGroup_AccessNil_DoesNotRevokeUserSessions(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{
				ID:   1,
				Name: "Dev",
				DatabaseAccesses: []domain.DatabaseAccess{
					{DatabaseID: 1, AccessType: domain.AccessViewOnly},
				},
			}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return g, nil
		},
	}
	sessionRevoked := false
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, _ []uint) error {
			sessionRevoked = true
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, sessionSvc)
	newName := "Renamed Group"

	// Act
	_, _ = service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{
		Name:             &newName,
		DatabaseAccesses: nil, // Only rename, no access change
	})

	// Assert
	if sessionRevoked {
		t.Fatal("expected sessions NOT to be revoked when DatabaseAccesses is nil")
	}
}

func TestUpdateGroup_ValidInput_ReturnsUpdatedGroup(t *testing.T) {
	// Arrange
	expectedGroup := domain.Group{
		ID:   1,
		Name: "New Name",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: 1, AccessType: domain.AccessViewOnly},
		},
	}
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Old Name"}, nil
		},
		UpdateFunc: func(_ context.Context, g domain.Group) (domain.Group, error) {
			return expectedGroup, nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})
	newName := "New Name"

	// Act
	updated, _ := service.UpdateGroup(context.Background(), 1, domain.UpdateGroupInput{Name: &newName})

	// Assert
	if !reflect.DeepEqual(updated, expectedGroup) {
		t.Fatalf("expected updated group %+v, got %+v", expectedGroup, updated)
	}
}

func TestDeleteGroup_GroupLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("group not found")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{}, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	err := service.DeleteGroup(context.Background(), 99)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteGroup_OrphanCheckDependencyFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("repository check orphan error")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	err := service.DeleteGroup(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteGroup_HasOrphanedUsers_ReturnsErrUserMustHaveGroup(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return true, nil // An orphaned user would result from deleting this group!
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	err := service.DeleteGroup(context.Background(), 1)

	// Assert
	if !errors.Is(err, domain.ErrUserMustHaveGroup) {
		t.Fatalf("expected ErrUserMustHaveGroup, got %v", err)
	}
}

func TestDeleteGroup_UserLookupFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to list user IDs")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return nil, expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	err := service.DeleteGroup(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteGroup_PersistenceFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("delete failed in database")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{}, nil
		},
		DeleteFunc: func(_ context.Context, _ uint) error {
			return expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, &testutil.SessionService{})

	// Act
	err := service.DeleteGroup(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteGroup_SessionRevocationFailure_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("failed to revoke sessions")
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		DeleteFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, _ []uint) error {
			return expectedErr
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, sessionSvc)

	// Act
	err := service.DeleteGroup(context.Background(), 1)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDeleteGroup_Success_DeletesGroup(t *testing.T) {
	// Arrange
	groupDeleted := false
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10}, nil
		},
		DeleteFunc: func(_ context.Context, id uint) error {
			if id == 1 {
				groupDeleted = true
			}
			return nil
		},
	}
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, _ []uint) error {
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, sessionSvc)

	// Act
	_ = service.DeleteGroup(context.Background(), 1)

	// Assert
	if !groupDeleted {
		t.Fatal("expected group to be deleted successfully")
	}
}

func TestDeleteGroup_Success_RevokesUserSessions(t *testing.T) {
	// Arrange
	groupRepo := &testutil.GroupRepository{
		FindByIDFunc: func(_ context.Context, _ uint) (domain.Group, error) {
			return domain.Group{ID: 1, Name: "Team"}, nil
		},
		HasUserWithOnlyThisGroupFunc: func(_ context.Context, _ uint) (bool, error) {
			return false, nil
		},
		FindAllUserIDsByGroupIDFunc: func(_ context.Context, _ uint) ([]uint, error) {
			return []uint{10, 20}, nil
		},
		DeleteFunc: func(_ context.Context, _ uint) error {
			return nil
		},
	}
	var capturedRevokedUserIDs []uint
	sessionSvc := &testutil.SessionService{
		RevokeUsersFunc: func(_ context.Context, userIDs []uint) error {
			capturedRevokedUserIDs = userIDs
			return nil
		},
	}
	service := groupservice.NewGroupService(groupRepo, groupRepo, &testutil.UserRepository{}, &testutil.DatabaseRepository{}, sessionSvc)

	// Act
	_ = service.DeleteGroup(context.Background(), 1)

	// Assert
	expectedUserIDs := []uint{10, 20}
	if !reflect.DeepEqual(capturedRevokedUserIDs, expectedUserIDs) {
		t.Fatalf("expected revoked users %v, got %v", expectedUserIDs, capturedRevokedUserIDs)
	}
}
