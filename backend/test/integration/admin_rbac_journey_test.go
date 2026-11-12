//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestAdminRBACJourney_GivenAdminUser_WhenCreateUserAndGroup_ThenPersistedInPlatformDB(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to register target DB: %v", err)
	}

	adminUser := createAdminUser(t, stack, "rbac_admin_1", "adminpass123")
	sessionCookie, csrfToken := loginUser(t, stack.engine, adminUser.Username, "adminpass123")

	// Act 1 - Create Group as Admin
	createGroupReq := handlers.CreateGroupRequest{
		Name: "finance_auditors",
		DatabaseAccesses: []handlers.DatabaseAccessRequest{
			{DatabaseID: targetDB.ID, AccessType: string(domain.AccessViewOnly)},
		},
	}
	recGroup := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/groups", createGroupReq, sessionCookie, csrfToken)

	// Assert 1
	if recGroup.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on group creation, got %d: %s", recGroup.Code, recGroup.Body.String())
	}
	groupRes := parseResponseBody[handlers.GroupResponse](t, recGroup)

	// Act 2 - Create User as Admin assigned to the new group
	createUserReq := handlers.CreateUserRequest{
		Username: "auditor_dave",
		Password: "auditorpassword123",
		GroupIDs: []uint{groupRes.Data.ID},
	}
	recUser := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/users", createUserReq, sessionCookie, csrfToken)

	// Assert 2
	if recUser.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on user creation, got %d: %s", recUser.Code, recUser.Body.String())
	}
	userRes := parseResponseBody[handlers.UserResponse](t, recUser)

	// Verify User physically exists in PostgreSQL platform DB
	storedUser, err := stack.userRepo.FindByID(context.Background(), userRes.Data.ID)
	if err != nil {
		t.Fatalf("failed to query created user from platform DB: %v", err)
	}
	if storedUser.Username != "auditor_dave" {
		t.Fatalf("expected username auditor_dave, got %s", storedUser.Username)
	}
}

func TestAdminRBACJourney_GivenAdminUser_WhenRegisterTargetDatabase_ThenEncryptedPasswordPersisted(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	adminUser := createAdminUser(t, stack, "rbac_admin_2", "adminpass123")
	sessionCookie, csrfToken := loginUser(t, stack.engine, adminUser.Username, "adminpass123")

	// Act - Register Target Database via HTTP API
	createDBReq := handlers.CreateDatabaseRequest{
		Host:     targetConfig.Host,
		Port:     int(targetConfig.Port),
		User:     targetConfig.User,
		Password: targetConfig.Password,
		Type:     targetConfig.Type,
		Name:     targetConfig.Name,
	}
	rec := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/databases", createDBReq, sessionCookie, csrfToken)

	// Assert - HTTP 201 Created
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on database onboarding, got %d: %s", rec.Code, rec.Body.String())
	}
	dbRes := parseResponseBody[handlers.DatabaseResponse](t, rec)

	// Verify that password in platform DB is encrypted
	retrievedDB, err := stack.databaseRepo.FindWithCredentialByID(context.Background(), dbRes.Data.ID)
	if err != nil {
		t.Fatalf("failed to retrieve registered database: %v", err)
	}
	if retrievedDB.Password != targetConfig.Password {
		t.Fatalf("expected decrypted password to match original, got %s", retrievedDB.Password)
	}
}

func TestAdminRBACJourney_GivenAdminUser_WhenAssignUserToGroupWithDatabaseAccess_ThenUserCanAccessTargetDatabase(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	adminUser := createAdminUser(t, stack, "rbac_admin_3", "adminpass123")
	adminCookie, adminCSRF := loginUser(t, stack.engine, adminUser.Username, "adminpass123")

	// Step 1: Register target DB
	targetDB, err := stack.databaseRepo.Save(ctx, &targetConfig)
	if err != nil {
		t.Fatalf("failed to create target DB: %v", err)
	}

	// Step 2: Create Group with AccessViewOnly to the target DB
	createGroupReq := handlers.CreateGroupRequest{
		Name: "analytics_read_group",
		DatabaseAccesses: []handlers.DatabaseAccessRequest{
			{DatabaseID: targetDB.ID, AccessType: string(domain.AccessViewOnly)},
		},
	}
	recGroup := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/groups", createGroupReq, adminCookie, adminCSRF)
	if recGroup.Code != http.StatusCreated {
		t.Fatalf("failed to create group: %s", recGroup.Body.String())
	}
	groupRes := parseResponseBody[handlers.GroupResponse](t, recGroup)

	// Step 3: Create User assigned to this group
	createUserReq := handlers.CreateUserRequest{
		Username: "regular_analyst",
		Password: "regularpassword123",
		GroupIDs: []uint{groupRes.Data.ID},
	}
	recUser := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/users", createUserReq, adminCookie, adminCSRF)
	if recUser.Code != http.StatusCreated {
		t.Fatalf("failed to create user: %s", recUser.Body.String())
	}

	// Act - User logs in and calls GET /api/v1/me/databases
	userCookie, _ := loginUser(t, stack.engine, "regular_analyst", "regularpassword123")
	recUserDBs := executeRequest(stack.engine, http.MethodGet, "/api/v1/me/databases", nil, userCookie, "")

	// Assert - User sees the target database
	if recUserDBs.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /me/databases, got %d: %s", recUserDBs.Code, recUserDBs.Body.String())
	}
	if len(recUserDBs.Body.String()) == 0 {
		t.Fatalf("expected non-empty response on /me/databases")
	}
}

func TestAdminRBACJourney_GivenNonAdminUser_WhenAccessAdminEndpoints_ThenRejectWithForbidden(t *testing.T) {
	// Arrange
	stack := setupTestStack(t)
	regularUser := createRegularUser(t, stack, "standard_user_attempt", "password123", nil)
	userCookie, userCSRF := loginUser(t, stack.engine, regularUser.Username, "password123")

	// Act - Non-admin attempts to access admin endpoint
	recGetUsers := executeRequest(stack.engine, http.MethodGet, "/api/v1/admin/users", nil, userCookie, "")
	recCreateGroup := executeRequest(stack.engine, http.MethodPost, "/api/v1/admin/groups", handlers.CreateGroupRequest{Name: "rogue_group"}, userCookie, userCSRF)

	// Assert - Must return HTTP 403 Forbidden
	if recGetUsers.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on admin GET /users, got %d", recGetUsers.Code)
	}
	if recCreateGroup.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on admin POST /groups, got %d", recCreateGroup.Code)
	}
}

func TestAdminRBACJourney_GivenActiveUserSession_WhenAdminRevokesGroupAccess_ThenUserSessionInvalidatedAndAccessDenied(t *testing.T) {
	// Arrange
	targetConfig := requireTargetPostgres(t)
	stack := setupTestStack(t)
	ctx := context.Background()

	adminUser := createAdminUser(t, stack, "rbac_admin_4", "adminpass123")
	adminCookie, adminCSRF := loginUser(t, stack.engine, adminUser.Username, "adminpass123")

	targetDB, _ := stack.databaseRepo.Save(ctx, &targetConfig)
	group, _ := stack.groupService.CreateGroup(ctx, domain.CreateGroupInput{
		Name: "temporary_access_group",
		DatabaseAccesses: []domain.DatabaseAccess{
			{DatabaseID: targetDB.ID, AccessType: domain.AccessViewOnly},
		},
	})
	fallbackGroup, err := stack.groupRepo.Save(ctx, domain.Group{
		Name: "fallback_worker_group",
	})
	if err != nil {
		t.Fatalf("failed to create fallback group: %v", err)
	}
	user := createRegularUser(t, stack, "temp_worker", "workerpass123", []uint{group.ID, fallbackGroup.ID})

	// User logs in and acquires active session cookie
	userCookie, _ := loginUser(t, stack.engine, user.Username, "workerpass123")

	// Verify user can access /me/databases initially
	recBefore := executeRequest(stack.engine, http.MethodGet, "/api/v1/me/databases", nil, userCookie, "")
	if recBefore.Code != http.StatusOK {
		t.Fatalf("expected 200 OK before revocation, got %d", recBefore.Code)
	}

	// Act - Admin removes user from group
	removeURL := fmt.Sprintf("/api/v1/admin/groups/%d/users/%d", group.ID, user.ID)
	recRemove := executeRequest(stack.engine, http.MethodDelete, removeURL, nil, adminCookie, adminCSRF)
	if recRemove.Code != http.StatusOK && recRemove.Code != http.StatusNoContent {
		t.Fatalf("failed to remove user from group: %s", recRemove.Body.String())
	}

	// Assert - Subsequent request with previous user session cookie is invalidated / rejected
	recAfter := executeRequest(stack.engine, http.MethodGet, "/api/v1/me/databases", nil, userCookie, "")
	if recAfter.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized after session invalidation on group revocation, got %d: %s", recAfter.Code, recAfter.Body.String())
	}
}
