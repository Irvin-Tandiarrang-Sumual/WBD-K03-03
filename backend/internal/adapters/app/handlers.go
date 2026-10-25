package app

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databaseexecutorhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_executor"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
)

type applicationHandlers struct {
	auth             *authhandler.AuthHandler
	user             *userhandler.UserHandler
	group            *grouphandler.GroupHandler
	database         *databasehandler.DatabaseHandler
	databaseExecutor *databaseexecutorhandler.DatabaseExecutorHandler
	migration        *migrationhandler.MigrationHandler
}

func buildHandlers(configuration *config.Config, services *applicationServices) *applicationHandlers {
	return &applicationHandlers{
		auth:             authhandler.NewAuthHandler(services.auth, configuration.Session.CookieSecure),
		user:             userhandler.NewUserHandler(services.user),
		group:            grouphandler.NewGroupHandler(services.group),
		database:         databasehandler.NewDatabaseHandler(services.database),
		databaseExecutor: databaseexecutorhandler.NewDatabaseExecutorHandler(services.databaseExecutor),
		migration:        migrationhandler.NewMigrationHandler(services.migration),
	}
}
