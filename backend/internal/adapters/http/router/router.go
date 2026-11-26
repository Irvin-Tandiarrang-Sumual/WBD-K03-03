package router

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type routerDependencies struct {
	authMiddleware        *middleware.AuthMiddleware
	csrfMiddleware        *middleware.CSRFMiddleware
	authHandler           *handlers.AuthHandler
	userHandler           *handlers.UserHandler
	groupHandler          *handlers.GroupHandler
	userGroupHandler      *handlers.UserGroupHandler
	databaseHandler       *handlers.DatabaseHandler
	databaseDriverHandler *handlers.DatabaseDriverHandler
	migrationHandler      *handlers.MigrationHandler
	draftHandler          *handlers.DraftHandler
	healthHandler         *handlers.HealthHandler
}

func SetupRouter(
	engine *gin.Engine,
	authMiddleware *middleware.AuthMiddleware,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	groupHandler *handlers.GroupHandler,
	userGroupHandler *handlers.UserGroupHandler,
	databaseHandler *handlers.DatabaseHandler,
	databaseDriverHandler *handlers.DatabaseDriverHandler,
	migrationHandler *handlers.MigrationHandler,
	draftHandler *handlers.DraftHandler,
	healthHandler *handlers.HealthHandler,
	sessionService ports.SessionService,
	userService ports.UserService,
	csrfMiddleware *middleware.CSRFMiddleware,
) {

	dependencies := routerDependencies{
		authMiddleware:        authMiddleware,
		csrfMiddleware:        csrfMiddleware,
		authHandler:           authHandler,
		userHandler:           userHandler,
		groupHandler:          groupHandler,
		userGroupHandler:      userGroupHandler,
		databaseHandler:       databaseHandler,
		databaseDriverHandler: databaseDriverHandler,
		migrationHandler:      migrationHandler,
		draftHandler:          draftHandler,
		healthHandler:         healthHandler,
	}

	engine.Use(middleware.Logger())
	engine.Use(middleware.ErrorHandler())

	api := engine.Group("/api/v1")
	registerHealthRoutes(api, dependencies)

	protected := api.Group("")
	protected.Use(
		dependencies.authMiddleware.Authenticate(),
		dependencies.csrfMiddleware.Validate(),
	)

	registerAuthRoutes(api, protected, dependencies)
	registerAdminRoutes(protected, dependencies)
	registerAuthenticatedUserRoutes(protected, dependencies)

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
