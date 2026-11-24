package router_test

import (
	"net/http"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestSetupRouter_ValidDependencies_RegistersRoutesWithoutPanic(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	sessionSvc := &testutil.SessionService{}
	userSvc := &testutil.UserService{}

	authMiddleware := middleware.NewAuthMiddleware(sessionSvc, userSvc, false)
	csrfMiddleware := middleware.NewCSRFMiddleware("http://localhost:3000")

	authHandler := handlers.NewAuthHandler(nil, false, userSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	groupHandler := handlers.NewGroupHandler(&testutil.GroupService{})
	userGroupHandler := handlers.NewUserGroupHandler(&testutil.UserGroupService{})
	databaseHandler := handlers.NewDatabaseHandler(&testutil.DatabaseService{})
	databaseDriverHandler := handlers.NewDatabaseDriverHandler(nil)
	migrationHandler := handlers.NewMigrationHandler(nil)
	draftHandler := handlers.NewDraftHandler(&testutil.DraftService{})
	healthHandler := handlers.NewHealthHandler(nil)

	// Act
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
		sessionSvc,
		userSvc,
		csrfMiddleware,
	)

	// Assert
	registered := make(map[string]bool)
	for _, r := range engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	expectedRoutes := []string{
		http.MethodPost + " /api/v1/databases/:id/drafts",
		http.MethodGet + " /api/v1/databases/:id/drafts",
		http.MethodGet + " /api/v1/drafts/:id",
		http.MethodPost + " /api/v1/drafts/:id/approvals",
		http.MethodPost + " /api/v1/me/databases/:database_id/drafts",
		http.MethodGet + " /api/v1/me/databases/:database_id/drafts",
		http.MethodGet + " /api/v1/me/drafts/:id",
		http.MethodPost + " /api/v1/me/drafts/:id/approvals",
	}

	for _, expected := range expectedRoutes {
		if !registered[expected] {
			t.Fatalf("expected route %q to be registered", expected)
		}
	}
}
