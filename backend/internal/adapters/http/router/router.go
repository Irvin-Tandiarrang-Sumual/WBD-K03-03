package router

import (
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
)

func SetUpRouter(
	r *gin.Engine,
	auth *authhandler.AuthHandler,
	users *userhandler.UserHandler,
	groups *grouphandler.GroupHandler,
	databases *databasehandler.DatabaseHandler,
	databaseExecutor *databaseexecutorhandler.DatabaseExecutorHandler,
	migrations *migrationhandler.MigrationHandler,
	sessionService ports.SessionService,
	userService ports.UserService,
	groupService ports.GroupService,
) {
	api := r.Group("/api/v1")

	api.POST("/auth/login", auth.Login)
	api.POST("/auth/logout", auth.Logout)

	admin := api.Group("/admin")

	admin.Use(
		middleware.Authenticate(sessionService),
		middleware.RequireAdmin(userService),
	)

	adminUsers := admin.Group("/users")

	adminUsers.GET(
		"",
		users.List,
	)

	adminUsers.GET(
		"/:userId",
		users.Get,
	)

	adminUsers.POST(
		"",
		users.Create,
	)

	adminUsers.PATCH(
		"/:userId",
		users.Update,
	)

	adminUsers.DELETE(
		"/:userId",
		users.Delete,
	)

	adminUsers.PATCH(
		"/:userId/role",
		users.SetRole,
	)

	adminGroups := admin.Group("/groups")

	adminGroups.GET(
		"",
		groups.List,
	)

	adminGroups.GET(
		"/:groupId",
		groups.Get,
	)

	adminGroups.POST(
		"",
		groups.Create,
	)

	adminGroups.PATCH(
		"/:groupId",
		groups.Update,
	)

	adminGroups.DELETE(
		"/:groupId",
		groups.Delete,
	)

	adminGroups.POST(
		"/:groupId/users/:userId",
		groups.AddUser,
	)

	adminGroups.DELETE(
		"/:groupId/users/:userId",
		groups.RemoveUser,
	)

	adminDatabases := admin.Group("/databases")

	adminDatabases.GET(
		"",
		databases.GetAllDatabases,
	)

	adminDatabases.GET(
		"/:databaseId",
		databases.GetDatabaseByID,
	)

	adminDatabases.POST(
		"",
		databases.CreateDatabase,
	)

	adminDatabases.DELETE(
		"/:databaseId",
		databases.DeleteDatabase,
	)

	me := api.Group("/me")
	me.Use(middleware.Authenticate(sessionService))

	meDatabases := me.Group("/databases")
	meDatabases.GET(
		"",
		databases.GetDatabasesByUserID,
	)
	meDatabases.GET(
		"/:database_id",
		middleware.RequireDatabaseView(groupService),
		databases.GetDatabaseByIDAndUserID,
	)
	meDatabases.POST(
		"/:database_id/query",
		middleware.RequireDatabaseView(groupService),
		databaseExecutor.ExecuteDatabaseQuery,
	)
	meDatabases.POST(
		"/:database_id/explain",
		middleware.RequireDatabaseView(groupService),
		databaseExecutor.ExplainDatabaseQuery,
	)
	meDatabases.GET(
		"/:database_id/migrations",
		middleware.RequireDatabaseView(groupService),
		migrations.GetMigrationsByDatabaseID,
	)
	meDatabases.POST(
		"/:database_id/migrations",
		middleware.RequireDatabaseEdit(groupService),
		migrations.CreateMigration,
	)
	meDatabases.GET(
		"/:database_id/migrations/compare",
		middleware.RequireDatabaseView(groupService),
		migrations.CompareMigrations,
	)
	meDatabases.GET(
		"/:database_id/migrations/:migration_id",
		middleware.RequireDatabaseView(groupService),
		migrations.GetMigrationByIDAndDatabaseID,
	)
	meDatabases.POST(
		"/migrations/rollback",
		migrations.RollbackMigrations,
	)
}