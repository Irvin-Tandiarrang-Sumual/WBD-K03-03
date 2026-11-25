package router

import (
	"github.com/gin-gonic/gin"
)

func registerAuthenticatedUserRoutes(protected *gin.RouterGroup, dependencies routerDependencies) {
	me := protected.Group("/me")

	registerUserDatabaseRoutes(me, dependencies)
	registerDraftRoutes(protected, dependencies)
	me.GET("/drafts/:id", dependencies.draftHandler.GetDraftByID)
	me.POST("/drafts/:id/approvals", dependencies.draftHandler.SubmitApproval)
}

func registerDraftRoutes(group *gin.RouterGroup, dependencies routerDependencies) {
	group.POST("/databases/:id/drafts", dependencies.draftHandler.CreateDraft)
	group.GET("/databases/:id/drafts", dependencies.draftHandler.GetDraftsByDatabaseID)
	group.GET("/drafts/:id", dependencies.draftHandler.GetDraftByID)
	group.POST("/drafts/:id/approvals", dependencies.draftHandler.SubmitApproval)
}

func registerUserDatabaseRoutes(me *gin.RouterGroup, dependencies routerDependencies) {
	databases := me.Group("/databases")
	databases.GET("", dependencies.databaseHandler.GetDatabasesByUserID)
	databases.GET(
		"/:database_id",
		dependencies.databaseHandler.GetDatabaseByIDAndUserID,
	)
	databases.POST(
		"/:database_id/query",
		dependencies.databaseDriverHandler.ExecuteDatabaseQuery,
	)
	databases.POST(
		"/:database_id/explain",
		dependencies.databaseDriverHandler.ExplainDatabaseQuery,
	)
	databases.POST(
		"/:database_id/explain-analyze",
		dependencies.databaseDriverHandler.ExplainAnalyzeDatabaseQuery,
	)
	databases.GET(
		"/:database_id/migrations",
		dependencies.migrationHandler.GetMigrationsByDatabaseID,
	)
	databases.POST(
		"/:database_id/migrations",
		dependencies.migrationHandler.CreateMigration,
	)
	databases.GET(
		"/:database_id/migrations/compare",
		dependencies.migrationHandler.CompareMigrations,
	)
	databases.GET(
		"/:database_id/migrations/:migration_id",
		dependencies.migrationHandler.GetMigrationByIDAndDatabaseID,
	)
	databases.POST("/:database_id/rollback-draft", dependencies.migrationHandler.ProposeRollback)
	databases.POST("/:database_id/forward-draft", dependencies.migrationHandler.ProposeForward)
	databases.POST("/:database_id/migration-drafts/:draft_id/vote", dependencies.migrationHandler.VoteMigrationDraft)
	databases.POST("/:database_id/migration-drafts/:draft_id/execute", dependencies.migrationHandler.ExecuteMigrationDraft)
	databases.POST(
		"/:database_id/drafts",
		dependencies.draftHandler.CreateDraft,
	)
	databases.GET(
		"/:database_id/drafts",
		dependencies.draftHandler.GetDraftsByDatabaseID,
	)
	databases.POST(
		"/migrations/rollback",
		dependencies.migrationHandler.RollbackMigrations,
	)
}
