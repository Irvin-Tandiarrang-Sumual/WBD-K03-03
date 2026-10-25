package router

import "github.com/gin-gonic/gin"

func registerHealthRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	api.GET("/health", dependencies.healthHandler.Check)
}
