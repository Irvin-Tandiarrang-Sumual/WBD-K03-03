//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	driver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/channel"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	databasedriverservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_driver"
	draftservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/draft"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	usergroupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user_group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/hasher"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const testCredentialKey = "0123456789abcdef0123456789abcdef"

type testStack struct {
	engine                *gin.Engine
	db                    *gorm.DB
	cipher                ports.DatabaseCredentialCipher
	poolManager           *driver.DatabasePoolManager
	databaseDriver        ports.DatabaseDriver
	userRepo              ports.UserRepository
	groupRepo             ports.GroupRepository
	userGroupRepo         ports.UserGroupRepository
	databaseRepo          ports.DatabaseRepository
	sessionRepo           ports.SessionRepository
	migrationRepo         ports.MigrationRepository
	userService           ports.UserService
	sessionService        ports.SessionService
	groupService          ports.GroupService
	databaseService       ports.DatabaseService
	databaseDriverService ports.DatabaseDriverService
	migrationService      ports.MigrationService
	authService           ports.AuthService
}

func requirePlatformDB(t *testing.T) *gorm.DB {
	t.Helper()
	if !testcontainersReady || platformGormDB == nil {
		t.Skip("Platform PostgreSQL testcontainer is not available. Please ensure Docker is running.")
	}
	resetPlatformDB(t, platformGormDB)
	return platformGormDB
}

func requireTargetPostgres(t *testing.T) domain.Database {
	t.Helper()
	if !testcontainersReady || targetPostgresConfig.Host == "" {
		t.Skip("Target PostgreSQL testcontainer is not available. Please ensure Docker is running.")
	}
	return targetPostgresConfig
}

func requireTargetMySQL(t *testing.T) domain.Database {
	t.Helper()
	if !testcontainersReady || targetMySQLConfig.Host == "" {
		t.Skip("Target MySQL testcontainer is not available. Please ensure Docker is running.")
	}
	return targetMySQLConfig
}

func resetPlatformDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(`
		TRUNCATE TABLE
			sessions,
			user_groups,
			group_databases,
			migrations,
			users,
			groups,
			databases
		CASCADE;
	`).Error
	if err != nil {
		t.Fatalf("failed to truncate platform DB tables: %v", err)
	}
}

func setupTestStack(t *testing.T) *testStack {
	t.Helper()
	db := requirePlatformDB(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	cipher, err := security.NewAESGCMCipher([]byte(testCredentialKey))
	if err != nil {
		t.Fatalf("failed to create test AES-GCM cipher: %v", err)
	}

	poolManager := driver.NewDatabasePoolManager()
	t.Cleanup(func() {
		_ = poolManager.CloseAll()
	})

	databaseDriver := driver.NewDatabaseDriver(poolManager)
	connectionTester := driver.NewDatabaseConnectionTester()

	userRepo := repositories.NewUserRepository(db)
	groupRepo := repositories.NewGroupRepository(db)
	userGroupRepo := repositories.NewUserGroupRepository(db)
	databaseRepo := repositories.NewDatabaseRepository(db, cipher)
	sessionRepo := repositories.NewSessionRepository(db)
	migrationRepo := repositories.NewMigrationRepository(db)

	sessionService := sessionservice.NewSessionService(sessionRepo)
	authService := authservice.NewAuthService(userRepo, sessionService)
	userService := userservice.NewUserService(userRepo, groupRepo, sessionService, userGroupRepo)
	userGroupService := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionService)
	groupService := groupservice.NewGroupService(userGroupRepo, groupRepo, userRepo, databaseRepo, sessionService)
	databaseService := databaseservice.NewDatabaseService(databaseRepo, connectionTester, userGroupRepo, sessionService, poolManager)
	databaseDriverService := databasedriverservice.NewDatabaseDriverService(databaseService, databaseRepo, databaseDriver, userGroupService)
	migrationService := migrationservice.NewMigrationService(migrationRepo, databaseRepo, databaseDriver, userGroupService)

	cfg := &config.Config{
		Server: config.ServerConfig{
			FrontendOrigin: "http://localhost:3000",
		},
		Session: config.SessionConfig{
			CookieSecure: false,
		},
	}

	authMiddleware := middleware.NewAuthMiddleware(sessionService, userService, false)
	csrfMiddleware := middleware.NewCSRFMiddleware(cfg.Server.FrontendOrigin)

	authHandler := handlers.NewAuthHandler(authService, false, userService)
	userHandler := handlers.NewUserHandler(userService)
	groupHandler := handlers.NewGroupHandler(groupService)
	userGroupHandler := handlers.NewUserGroupHandler(userGroupService)
	databaseHandler := handlers.NewDatabaseHandler(databaseService)
	databaseDriverHandler := handlers.NewDatabaseDriverHandler(databaseDriverService)
	migrationHandler := handlers.NewMigrationHandler(migrationService)
	draftHandler := newTestDraftHandler(t, db, databaseRepo, groupRepo, migrationRepo, databaseDriver, userGroupService)
	healthHandler := handlers.NewHealthHandler(db)

	router.SetupRouter(
		engine,
		authMiddleware,
		authHandler,
		userHandler,
		groupHandler,
		userGroupHandler,
		databaseHandler,
		databaseDriverHandler,
		migrationHandler,
		draftHandler,
		healthHandler,
		sessionService,
		userService,
		csrfMiddleware,
	)

	return &testStack{
		engine:                engine,
		db:                    db,
		cipher:                cipher,
		poolManager:           poolManager,
		databaseDriver:        databaseDriver,
		userRepo:              userRepo,
		groupRepo:             groupRepo,
		userGroupRepo:         userGroupRepo,
		databaseRepo:          databaseRepo,
		sessionRepo:           sessionRepo,
		migrationRepo:         migrationRepo,
		userService:           userService,
		sessionService:        sessionService,
		groupService:          groupService,
		databaseService:       databaseService,
		databaseDriverService: databaseDriverService,
		migrationService:      migrationService,
		authService:           authService,
	}
}

func newTestDraftHandler(
	t *testing.T,
	db *gorm.DB,
	databaseRepo ports.DatabaseRepository,
	groupRepo ports.GroupRepository,
	migrationRepo ports.MigrationRepository,
	databaseDriver ports.DatabaseDriver,
	userGroupService ports.UserGroupService,
) *handlers.DraftHandler {
	t.Helper()
	draftRepo := repositories.NewDraftRepository(db)
	var service ports.DraftService
	worker := draftservice.NewExecutionWorker(
		draftRepo, databaseRepo, migrationRepo, databaseDriver, groupRepo,
		func(databaseID uint) { service.UnlockDatabase(databaseID) },
		ports.NoopMigrationMetrics{},
		userGroupService,
	)
	queue := channel.NewChannelQueue(worker.Execute, 2, 16, channel.DefaultTaskTimeout)
	t.Cleanup(queue.Close)
	service = draftservice.NewDraftService(draftRepo, groupRepo, userGroupService, databaseRepo, queue, migrationRepo, databaseDriver)
	return handlers.NewDraftHandler(service)
}

func createAdminUser(t *testing.T, stack *testStack, username, password string) domain.User {
	t.Helper()
	return createUserWithOwnGroup(t, stack, username, password, domain.RoleAdmin, "admin_group_"+username)
}

func createUserWithOwnGroup(t *testing.T, stack *testStack, username, password, role, groupName string) domain.User {
	t.Helper()
	user := insertTestUser(t, stack, username, password, role)
	saveTestGroup(t, stack, groupName, user.ID)
	return domain.User{ID: user.ID, Username: user.Username, Role: role}
}

func createTestGroup(t *testing.T, stack *testStack, name string) domain.Group {
	t.Helper()
	leader := createMigrationLeader(t, stack, "leader_"+name)
	return saveTestGroup(t, stack, name, leader.ID)
}

func saveTestGroup(t *testing.T, stack *testStack, name string, leaderID uint) domain.Group {
	t.Helper()
	group, err := stack.groupRepo.Save(context.Background(), domain.Group{Name: name, LeaderID: leaderID})
	if err != nil {
		t.Fatalf("failed to create group %s: %v", name, err)
	}
	return group
}

func createRegularUser(t *testing.T, stack *testStack, username, password string, groupIDs []uint) domain.User {
	t.Helper()
	if len(groupIDs) == 0 {
		return createUserWithOwnGroup(t, stack, username, password, domain.RoleUser, "default_group_"+username)
	}
	hashed, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := domain.User{
		Username: username,
		Password: hashed,
		Role:     domain.RoleUser,
	}
	created, err := stack.userRepo.Save(context.Background(), user, groupIDs)
	if err != nil {
		t.Fatalf("failed to create regular user: %v", err)
	}
	return created
}

func loginUser(t *testing.T, engine *gin.Engine, username, password string) (sessionCookie string, csrfToken string) {
	t.Helper()
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed for %s, expected 200 got %d: %s", username, rec.Code, rec.Body.String())
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == middleware.SessionCookieName {
			sessionCookie = c.Value
			break
		}
	}
	if sessionCookie == "" {
		t.Fatalf("login response did not set session cookie")
	}

	var res struct {
		Data struct {
			CSRFToken string `json:"csrfToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	csrfToken = res.Data.CSRFToken
	if csrfToken == "" {
		var resSnake struct {
			Data struct {
				CSRFToken string `json:"csrf_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resSnake)
		csrfToken = resSnake.Data.CSRFToken
	}

	return sessionCookie, csrfToken
}

func executeRequest(
	engine *gin.Engine,
	method string,
	urlPath string,
	body any,
	sessionCookie string,
	csrfToken string,
) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, urlPath, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: sessionCookie,
		})
	}
	if csrfToken != "" {
		req.Header.Set(middleware.CSRFHeaderName, csrfToken)
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func parseResponseBody[T any](t *testing.T, rec *httptest.ResponseRecorder) response.BaseResponse[T] {
	t.Helper()
	var res response.BaseResponse[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err == nil && res.Success {
		return res
	}
	var direct T
	if err := json.Unmarshal(rec.Body.Bytes(), &direct); err == nil {
		return response.BaseResponse[T]{Success: true, Data: direct}
	}
	t.Fatalf("failed to decode response JSON: raw body: %s", rec.Body.String())
	return res
}

func createMigrationLeader(t *testing.T, stack *testStack, username string) model.User {
	t.Helper()
	return insertTestUser(t, stack, username, "password123", domain.RoleUser)
}

func insertTestUser(t *testing.T, stack *testStack, username, password, role string) model.User {
	t.Helper()
	hash, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	leader := model.User{Username: username, Password: hash, Role: role}
	if err := stack.db.Session(&gorm.Session{SkipHooks: true}).Create(&leader).Error; err != nil {
		t.Fatal(err)
	}
	return leader
}

type testSession struct {
	cookie string
	csrf   string
}

func signIn(t *testing.T, stack *testStack, username string) testSession {
	t.Helper()
	cookie, csrf := loginUser(t, stack.engine, username, "password123")
	return testSession{cookie: cookie, csrf: csrf}
}

func applyTestMigration(t *testing.T, stack *testStack, databaseID uint, author testSession, sqlUp, sqlDown string, leaders ...testSession) handlers.MigrationResponse {
	t.Helper()
	base := fmt.Sprintf("/api/v1/me/databases/%d", databaseID)
	request := handlers.CreateDraftRequest{SqlUp: sqlUp, SqlDown: sqlDown}
	proposal := executeRequest(stack.engine, http.MethodPost, base+"/migrations", request, author.cookie, author.csrf)
	if proposal.Code != http.StatusCreated {
		t.Fatalf("expected 201 for migration draft: %s", proposal.Body.String())
	}
	draft := parseResponseBody[handlers.DraftResponse](t, proposal).Data
	if draft.Status != domain.DraftStatusPendingApproval {
		t.Fatalf("expected draft to await approval, got %s", draft.Status)
	}
	path := fmt.Sprintf("/api/v1/drafts/%d", draft.ID)
	early := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, author.cookie, author.csrf)
	if early.Code != http.StatusConflict {
		t.Fatalf("expected unapproved execution to be rejected: %s", early.Body.String())
	}
	for _, leader := range leaders {
		approveTestDraft(t, stack, path, leader)
	}
	accepted := executeRequest(stack.engine, http.MethodPost, path+"/execute", nil, author.cookie, author.csrf)
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("expected 202 after approval: %s", accepted.Body.String())
	}
	job := parseResponseBody[handlers.DraftExecutionResponse](t, accepted).Data
	if job.JobID == "" || job.DraftID != draft.ID {
		t.Fatalf("expected job ID for draft %d, got %+v", draft.ID, job)
	}
	return waitForAppliedMigration(t, stack, databaseID, draft.ID, author)
}

func approveTestDraft(t *testing.T, stack *testStack, path string, voter testSession) {
	t.Helper()
	vote := executeRequest(stack.engine, http.MethodPost, path+"/approvals",
		handlers.SubmitApprovalRequest{Status: domain.VoteApproved}, voter.cookie, voter.csrf)
	if vote.Code != http.StatusOK {
		t.Fatalf("expected 200 for approval: %s", vote.Body.String())
	}
}

func waitForAppliedMigration(t *testing.T, stack *testStack, databaseID, draftID uint, author testSession) handlers.MigrationResponse {
	t.Helper()
	path := fmt.Sprintf("/api/v1/drafts/%d", draftID)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		rec := executeRequest(stack.engine, http.MethodGet, path, nil, author.cookie, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("failed to fetch draft: %s", rec.Body.String())
		}
		status := parseResponseBody[handlers.DraftResponse](t, rec).Data.Status
		switch status {
		case domain.DraftStatusFailed, domain.DraftStatusInvalidated:
			t.Fatalf("draft %d ended in %s", draftID, status)
		case domain.DraftStatusApplied:
			return getDraftMigration(t, stack, databaseID, draftID, author)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("draft %d did not finish", draftID)
	return handlers.MigrationResponse{}
}

func getDraftMigration(t *testing.T, stack *testStack, databaseID, draftID uint, author testSession) handlers.MigrationResponse {
	t.Helper()
	var migration model.Migration
	if err := stack.db.Select("id").Where("database_id = ? AND draft_id = ?", databaseID, draftID).First(&migration).Error; err != nil {
		t.Fatalf("failed to find migration for draft %d: %v", draftID, err)
	}
	path := fmt.Sprintf("/api/v1/me/databases/%d/migrations/%d", databaseID, migration.ID)
	rec := executeRequest(stack.engine, http.MethodGet, path, nil, author.cookie, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("failed to fetch migration: %s", rec.Body.String())
	}
	return parseResponseBody[handlers.MigrationResponse](t, rec).Data
}
