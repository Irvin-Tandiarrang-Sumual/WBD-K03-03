package router

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func registerAdminRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	admin := api.Group("/admin")
	admin.Use(
		dependencies.authMiddleware.Authenticate(),
		middleware.RequireAdmin(),
	)

	registerAdminUserRoutes(admin, dependencies)
	registerAdminGroupRoutes(admin, dependencies)
	registerAdminUserGroupRoutes(admin, dependencies)
	registerAdminDatabaseRoutes(admin, dependencies)
}

func registerAdminUserRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	users := admin.Group("/users")
	users.GET("", dependencies.userHandler.GetAllUsers)
	users.GET("/:id", dependencies.userHandler.GetUserByID)
	users.POST("", dependencies.userHandler.CreateUser)
	users.PATCH("/:id", dependencies.userHandler.UpdateUser)
	users.DELETE("/:id", dependencies.userHandler.DeleteUser)
	users.PATCH("/:id/role", dependencies.userHandler.UpdateUserRole)
}

func registerAdminGroupRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	groups := admin.Group("/groups")
	groups.GET("", dependencies.groupHandler.GetAllGroups)
	groups.GET("/:id", dependencies.groupHandler.GetGroupByID)
	groups.POST("", dependencies.groupHandler.CreateGroup)
	groups.PATCH("/:id", dependencies.groupHandler.UpdateGroup)
	groups.DELETE("/:id", dependencies.groupHandler.DeleteGroup)
}

func registerAdminUserGroupRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	groups := admin.Group("/groups")
	if dependencies.userGroupHandler != nil {
		groups.POST("/:group_id/users/:user_id", dependencies.userGroupHandler.AddUser)
		groups.DELETE("/:group_id/users/:user_id", dependencies.userGroupHandler.RemoveUser)
	}
}

func registerAdminDatabaseRoutes(admin *gin.RouterGroup, dependencies routerDependencies) {
	databases := admin.Group("/databases")
	databases.GET("", dependencies.databaseHandler.GetAllDatabases)
	databases.GET("/:id", dependencies.databaseHandler.GetDatabaseByID)
	databases.POST("", dependencies.databaseHandler.CreateDatabase)
	databases.DELETE("/:id", dependencies.databaseHandler.DeleteDatabase)
}
