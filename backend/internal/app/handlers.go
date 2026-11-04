package app

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	databasedriverhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database_driver"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	healthhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/health"
	migrationhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/migration"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"
	usergrouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user_group"
	"gorm.io/gorm"
)

type applicationHandlers struct {
	auth           *authhandler.AuthHandler
	user           *userhandler.UserHandler
	group          *grouphandler.GroupHandler
	userGroup      *usergrouphandler.UserGroupHandler
	database       *databasehandler.DatabaseHandler
	databaseDriver *databasedriverhandler.DatabaseDriverHandler
	migration      *migrationhandler.MigrationHandler
	health         *healthhandler.HealthHandler
}

func buildHandlers(configuration *config.Config, services *applicationServices, systemDatabase *gorm.DB) *applicationHandlers {
	return &applicationHandlers{
		auth:           authhandler.NewAuthHandler(services.auth, configuration.Session.CookieSecure, services.user),
		user:           userhandler.NewUserHandler(services.user),
		group:          grouphandler.NewGroupHandler(services.group),
		userGroup:      usergrouphandler.NewUserGroupHandler(services.userGroup),
		database:       databasehandler.NewDatabaseHandler(services.database),
		databaseDriver: databasedriverhandler.NewDatabaseDriverHandler(services.databaseDriver),
		migration:      migrationhandler.NewMigrationHandler(services.migration),
		health:         healthhandler.NewHealthHandler(systemDatabase),
	}
}
