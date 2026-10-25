package router

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func registerAuthenticatedUserRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	me := api.Group("/me")
	me.Use(middleware.Authenticate(dependencies.sessionService))

	registerUserDatabaseRoutes(me, dependencies)
}

func registerUserDatabaseRoutes(me *gin.RouterGroup, dependencies routerDependencies) {
	databases := me.Group("/databases")
	databases.GET("", dependencies.databaseHandler.GetDatabasesByUserID)
	databases.GET(
		"/:database_id",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.databaseHandler.GetDatabaseByIDAndUserID,
	)
	databases.POST(
		"/:database_id/query",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.databaseExecutorHandler.ExecuteDatabaseQuery,
	)
	databases.POST(
		"/:database_id/explain",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.databaseExecutorHandler.ExplainDatabaseQuery,
	)
	databases.GET(
		"/:database_id/migrations",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.migrationHandler.GetMigrationsByDatabaseID,
	)
	databases.POST(
		"/:database_id/migrations",
		middleware.RequireDatabaseEdit(dependencies.groupService),
		dependencies.migrationHandler.CreateMigration,
	)
	databases.GET(
		"/:database_id/migrations/compare",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.migrationHandler.CompareMigrations,
	)
	databases.GET(
		"/:database_id/migrations/:migration_id",
		middleware.RequireDatabaseView(dependencies.groupService),
		dependencies.migrationHandler.GetMigrationByIDAndDatabaseID,
	)
	databases.POST(
		"/migrations/rollback",
		dependencies.migrationHandler.RollbackMigrations,
	)
}
