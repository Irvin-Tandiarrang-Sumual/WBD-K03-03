package router

import (
	"github.com/gin-gonic/gin"
)

func registerAuthRoutes(public *gin.RouterGroup, protected *gin.RouterGroup, dependencies routerDependencies) {
	public.POST("/auth/login", dependencies.authHandler.Login)
	protected.POST("/auth/logout", dependencies.authHandler.Logout)
	protected.GET("/auth/me", dependencies.authHandler.Me)
}
