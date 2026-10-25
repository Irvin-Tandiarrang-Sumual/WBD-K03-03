package router

import "github.com/gin-gonic/gin"

func registerAuthRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	api.POST("/auth/login", dependencies.authHandler.Login)
	api.POST("/auth/logout", dependencies.authHandler.Logout)
}
