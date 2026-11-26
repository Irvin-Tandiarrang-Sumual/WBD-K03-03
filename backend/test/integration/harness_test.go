//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
