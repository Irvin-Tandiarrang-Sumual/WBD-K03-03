package router

import (
	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type routerDependencies struct {
	authHandler             *authhandler.AuthHandler
	userHandler             *userhandler.UserHandler
	groupHandler            *grouphandler.GroupHandler
	databaseHandler         *databasehandler.DatabaseHandler
	databaseExecutorHandler *databaseexecutorhandler.DatabaseExecutorHandler
	migrationHandler        *migrationhandler.MigrationHandler
	sessionService          ports.SessionService
	userService             ports.UserService
	groupService            ports.GroupService
}

func SetupRouter(
	engine *gin.Engine,
	authHandler *authhandler.AuthHandler,
	userHandler *userhandler.UserHandler,
	groupHandler *grouphandler.GroupHandler,
	databaseHandler *databasehandler.DatabaseHandler,
	databaseExecutorHandler *databaseexecutorhandler.DatabaseExecutorHandler,
	migrationHandler *migrationhandler.MigrationHandler,
	sessionService ports.SessionService,
	userService ports.UserService,
	groupService ports.GroupService,
) {
	dependencies := routerDependencies{
		authHandler:             authHandler,
		userHandler:             userHandler,
		groupHandler:            groupHandler,
		databaseHandler:         databaseHandler,
		databaseExecutorHandler: databaseExecutorHandler,
		migrationHandler:        migrationHandler,
		sessionService:          sessionService,
		userService:             userService,
		groupService:            groupService,
	}

	api := engine.Group("/api/v1")
	registerAuthRoutes(api, dependencies)
	registerAdminRoutes(api, dependencies)
	registerAuthenticatedUserRoutes(api, dependencies)
}
