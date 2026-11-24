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
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
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
	draftRepo             ports.DraftRepository
	userService           ports.UserService
	sessionService        ports.SessionService
	groupService          ports.GroupService
	databaseService       ports.DatabaseService
	databaseDriverService ports.DatabaseDriverService
	migrationService      ports.MigrationService
	draftService          ports.DraftService
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
			draft_checkpoints,
			draft_approvals,
			drafts,
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
	draftRepo := repositories.NewDraftRepository(db)

	sessionService := sessionservice.NewSessionService(sessionRepo)
	authService := authservice.NewAuthService(userRepo, sessionService)
	userService := userservice.NewUserService(userRepo, groupRepo, sessionService, userGroupRepo)
	userGroupService := usergroupservice.NewUserGroupService(userGroupRepo, groupRepo, userRepo, sessionService)
	groupService := groupservice.NewGroupService(userGroupRepo, groupRepo, userRepo, databaseRepo, sessionService)
	databaseService := databaseservice.NewDatabaseService(databaseRepo, connectionTester, userGroupRepo, sessionService, poolManager)
	databaseDriverService := databasedriverservice.NewDatabaseDriverService(databaseService, databaseRepo, databaseDriver, userGroupService)
	migrationService := migrationservice.NewMigrationService(migrationRepo, databaseRepo, databaseDriver, userGroupService)
	draftService := draftservice.NewDraftService(draftRepo, groupRepo, userGroupService)

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
	draftHandler := handlers.NewDraftHandler(draftService)
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
		draftRepo:             draftRepo,
		userService:           userService,
		sessionService:        sessionService,
		groupService:          groupService,
		databaseService:       databaseService,
		databaseDriverService: databaseDriverService,
		migrationService:      migrationService,
		draftService:          draftService,
		authService:           authService,
	}
}

func createAdminUser(t *testing.T, stack *testStack, username, password string) domain.User {
	t.Helper()
	hashed, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	adminGroup, err := stack.groupRepo.Save(context.Background(), domain.Group{
		Name: "admin_group_" + username,
	})
	if err != nil {
		t.Fatalf("failed to create admin group: %v", err)
	}
	user := domain.User{
		Username: username,
		Password: hashed,
		Role:     domain.RoleAdmin,
	}
	created, err := stack.userRepo.Save(context.Background(), user, []uint{adminGroup.ID})
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	return created
}

func createRegularUser(t *testing.T, stack *testStack, username, password string, groupIDs []uint) domain.User {
	t.Helper()
	hashed, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if len(groupIDs) == 0 {
		defaultGroup, err := stack.groupRepo.Save(context.Background(), domain.Group{
			Name: "default_group_" + username,
		})
		if err != nil {
			t.Fatalf("failed to create default group: %v", err)
		}
		groupIDs = []uint{defaultGroup.ID}
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
