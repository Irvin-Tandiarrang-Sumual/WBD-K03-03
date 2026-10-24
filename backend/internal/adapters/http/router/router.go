package router

import (
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
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
	sessionService ports.SessionService,
	userService ports.UserService,
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
	meDatabases.POST(
		"/:database_id/query",
		databaseExecutor.ExecuteDatabaseQuery,
	)
	meDatabases.POST(
		"/:database_id/explain",
		databaseExecutor.ExplainDatabaseQuery,
	)
}