package router

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func registerAdminRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	admin := api.Group("/admin")
	admin.Use(
		middleware.Authenticate(dependencies.sessionService),
		middleware.RequireAdmin(dependencies.userService),
	)

	registerAdminUserRoutes(admin, dependencies)
	registerAdminGroupRoutes(admin, dependencies)
	registerAdminDatabaseRoutes(admin, dependencies)
}

func registerAdminUserRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	users := admin.Group("/users")
	users.GET("", dependencies.userHandler.List)
	users.GET("/:userId", dependencies.userHandler.Get)
	users.POST("", dependencies.userHandler.Create)
	users.PATCH("/:userId", dependencies.userHandler.Update)
	users.DELETE("/:userId", dependencies.userHandler.Delete)
	users.PATCH("/:userId/role", dependencies.userHandler.SetRole)
}

func registerAdminGroupRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	groups := admin.Group("/groups")
	groups.GET("", dependencies.groupHandler.List)
	groups.GET("/:groupId", dependencies.groupHandler.Get)
	groups.POST("", dependencies.groupHandler.Create)
	groups.PATCH("/:groupId", dependencies.groupHandler.Update)
	groups.DELETE("/:groupId", dependencies.groupHandler.Delete)
	groups.POST("/:groupId/users/:userId", dependencies.groupHandler.AddUser)
	groups.DELETE("/:groupId/users/:userId", dependencies.groupHandler.RemoveUser)
}

func registerAdminDatabaseRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	databases := admin.Group("/databases")
	databases.GET("", dependencies.databaseHandler.GetAllDatabases)
	databases.GET("/:id", dependencies.databaseHandler.GetDatabaseByID)
	databases.POST("", dependencies.databaseHandler.CreateDatabase)
	databases.DELETE("/:id", dependencies.databaseHandler.DeleteDatabase)
}
