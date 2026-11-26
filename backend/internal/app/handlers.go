package app

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers"
	queuepostgres "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/postgresql"
	"gorm.io/gorm"
)

type applicationHandlers struct {
	auth           *handlers.AuthHandler
	user           *handlers.UserHandler
	group          *handlers.GroupHandler
	userGroup      *handlers.UserGroupHandler
	database       *handlers.DatabaseHandler
	databaseDriver *handlers.DatabaseDriverHandler
	migration      *handlers.MigrationHandler
	draft          *handlers.DraftHandler
	health         *handlers.HealthHandler
	metrics        *handlers.MetricsHandler
}

func buildHandlers(configuration *config.Config, services *applicationServices, systemDatabase *gorm.DB, pools *driver.DatabasePoolManager) *applicationHandlers {
	return &applicationHandlers{
		auth:           handlers.NewAuthHandler(services.auth, configuration.Session.CookieSecure, services.user),
		user:           handlers.NewUserHandler(services.user),
		group:          handlers.NewGroupHandler(services.group),
		userGroup:      handlers.NewUserGroupHandler(services.userGroup),
		database:       handlers.NewDatabaseHandler(services.database),
		databaseDriver: handlers.NewDatabaseDriverHandler(services.databaseDriver),
		migration:      handlers.NewMigrationHandler(services.migration),
		draft:          handlers.NewDraftHandler(services.draft),
		health:         handlers.NewHealthHandler(systemDatabase),
		metrics: handlers.NewMetricsHandler(func(ctx context.Context) error {
			pools.CollectMetrics()
			return queuepostgres.CollectMetrics(ctx, systemDatabase)
		}),
	}
}
