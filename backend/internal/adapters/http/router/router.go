package router

import (
	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databasedriverhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_driver"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	healthhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/health"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	usergrouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user_group"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type routerDependencies struct {
	authMiddleware        *middleware.AuthMiddleware
	authHandler           *authhandler.AuthHandler
	userHandler           *userhandler.UserHandler
	groupHandler          *grouphandler.GroupHandler
	userGroupHandler      *usergrouphandler.UserGroupHandler
	databaseHandler       *databasehandler.DatabaseHandler
	databaseDriverHandler *databasedriverhandler.DatabaseDriverHandler
	migrationHandler      *migrationhandler.MigrationHandler
	healthHandler         *healthhandler.HealthHandler
}

func SetupRouter(
	engine *gin.Engine,
	authMiddleware *middleware.AuthMiddleware,
	authHandler *authhandler.AuthHandler,
	userHandler *userhandler.UserHandler,
	groupHandler *grouphandler.GroupHandler,
	userGroupHandler *usergrouphandler.UserGroupHandler,
	databaseHandler *databasehandler.DatabaseHandler,
	databaseDriverHandler *databasedriverhandler.DatabaseDriverHandler,
	migrationHandler *migrationhandler.MigrationHandler,
	healthHandler *healthhandler.HealthHandler,
	sessionService ports.SessionService,
	userService ports.UserService,
) {
	dependencies := routerDependencies{
		authMiddleware:        authMiddleware,
		authHandler:           authHandler,
		userHandler:           userHandler,
		groupHandler:          groupHandler,
		userGroupHandler:      userGroupHandler,
		databaseHandler:       databaseHandler,
		databaseDriverHandler: databaseDriverHandler,
		migrationHandler:      migrationHandler,
		healthHandler:         healthHandler,
	}

	api := engine.Group("/api/v1")
	registerHealthRoutes(api, dependencies)
	registerAuthRoutes(api, dependencies)
	registerAdminRoutes(api, dependencies)
	registerAuthenticatedUserRoutes(api, dependencies)

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
