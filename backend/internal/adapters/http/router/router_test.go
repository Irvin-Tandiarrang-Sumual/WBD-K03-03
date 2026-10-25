package router

import (
	"testing"

	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	healthhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/health"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestSetupRouterRegistersExpectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	SetupRouter(
		engine,
		authhandler.NewAuthHandler(nil, false),
		userhandler.NewUserHandler(&testutil.UserService{}),
		grouphandler.NewGroupHandler(&testutil.GroupService{}),
		databasehandler.NewDatabaseHandler(&testutil.DatabaseService{}),
		databaseexecutorhandler.NewDatabaseExecutorHandler(nil),
		migrationhandler.NewMigrationHandler(&testutil.MigrationService{}),
		healthhandler.NewHealthHandler(nil),
		&testutil.SessionService{},
		&testutil.UserService{},
		&testutil.GroupService{},
	)

	got := make(map[string]bool)
	for _, route := range engine.Routes() {
		got[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/logout",
		"GET /api/v1/admin/users",
		"GET /api/v1/admin/users/:userId",
		"POST /api/v1/admin/users",
		"PATCH /api/v1/admin/users/:userId",
		"DELETE /api/v1/admin/users/:userId",
		"PATCH /api/v1/admin/users/:userId/role",
		"GET /api/v1/admin/groups",
		"GET /api/v1/admin/groups/:groupId",
		"POST /api/v1/admin/groups",
		"PATCH /api/v1/admin/groups/:groupId",
		"DELETE /api/v1/admin/groups/:groupId",
		"POST /api/v1/admin/groups/:groupId/users/:userId",
		"DELETE /api/v1/admin/groups/:groupId/users/:userId",
		"GET /api/v1/admin/databases",
		"GET /api/v1/admin/databases/:id",
		"POST /api/v1/admin/databases",
		"DELETE /api/v1/admin/databases/:id",
		"GET /api/v1/me/databases",
		"GET /api/v1/me/databases/:database_id",
		"POST /api/v1/me/databases/:database_id/query",
		"POST /api/v1/me/databases/:database_id/explain",
		"GET /api/v1/me/databases/:database_id/migrations",
		"POST /api/v1/me/databases/:database_id/migrations",
		"GET /api/v1/me/databases/:database_id/migrations/compare",
		"GET /api/v1/me/databases/:database_id/migrations/:migration_id",
		"POST /api/v1/me/databases/migrations/rollback",
		"GET /api/v1/health",
		"GET /swagger/*any",
	}

	for _, route := range expected {
		if !got[route] {
			t.Errorf("route not registered: %s", route)
		}
	}
	if len(got) != len(expected) {
		t.Fatalf("registered route count=%d, want %d; routes=%v", len(got), len(expected), got)
	}
}
