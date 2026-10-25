package router

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func registerAuthRoutes(api *gin.RouterGroup, dependencies routerDependencies) {
	api.POST("/auth/login", dependencies.authHandler.Login)
	api.POST("/auth/logout", dependencies.authHandler.Logout)
	api.GET("/auth/me", middleware.Authenticate(dependencies.sessionService), dependencies.authHandler.Me)
}
