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
	me.POST("/drafts/:id/execute", dependencies.draftHandler.ExecuteDraft)
	me.POST("/drafts/:id/resume", dependencies.draftHandler.ResumeBatch)
	me.POST("/drafts/:id/revert-batch", dependencies.draftHandler.RevertBatch)
	me.GET("/drafts/:id/progress", dependencies.draftHandler.GetBatchProgress)
	me.GET("/drafts/:id/progress/stream", dependencies.draftHandler.StreamBatchProgress)
}

func registerDraftRoutes(group *gin.RouterGroup, dependencies routerDependencies) {
	group.POST("/databases/:id/drafts", dependencies.draftHandler.CreateDraft)
	group.GET("/databases/:id/drafts", dependencies.draftHandler.GetDraftsByDatabaseID)
	group.GET("/drafts/:id", dependencies.draftHandler.GetDraftByID)
	group.POST("/drafts/:id/approvals", dependencies.draftHandler.SubmitApproval)
	group.POST("/drafts/:id/execute", dependencies.draftHandler.ExecuteDraft)
	group.POST("/drafts/:id/resume", dependencies.draftHandler.ResumeBatch)
	group.POST("/drafts/:id/revert-batch", dependencies.draftHandler.RevertBatch)
	group.GET("/drafts/:id/progress", dependencies.draftHandler.GetBatchProgress)
	group.GET("/drafts/:id/progress/stream", dependencies.draftHandler.StreamBatchProgress)
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
	databases.POST("/:database_id/rollback-draft", dependencies.draftHandler.ProposeRollback)
	databases.POST("/:database_id/forward-draft", dependencies.draftHandler.ProposeForward)
	databases.POST("/:database_id/migration-drafts/:draft_id/vote", dependencies.draftHandler.VoteDraft)
	databases.POST("/:database_id/migration-drafts/:draft_id/execute", dependencies.draftHandler.ExecuteMigrationDraft)
	databases.POST(
		"/:database_id/drafts",
		dependencies.draftHandler.CreateDraft,
	)
	databases.GET(
		"/:database_id/drafts",
		dependencies.draftHandler.GetDraftsByDatabaseID,
	)
}
