package app

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/channel"
	queuepostgres "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/queue/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	databasedriverservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_driver"
	draftservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/draft"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	usergroupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user_group"
	"gorm.io/gorm"
)

type applicationServices struct {
	auth           ports.AuthService
	user           ports.UserService
	group          ports.GroupService
	userGroup      ports.UserGroupService
	session        ports.SessionService
	database       ports.DatabaseService
	databaseDriver ports.DatabaseDriverService
	migration      ports.MigrationService
	draft          ports.DraftService
	queue          *queuepostgres.Queue
}

func (s *applicationServices) close() {
	if s.queue != nil {
		s.queue.Close()
	}
}

func buildServices(
	systemDatabase *gorm.DB,
	infrastructure *targetDatabaseInfrastructure,
	workerConfig config.WorkerConfig,
) (*applicationServices, error) {
	userRepository := repositories.NewUserRepository(systemDatabase)
	groupRepository := repositories.NewGroupRepository(systemDatabase)
	userGroupRepository := repositories.NewUserGroupRepository(systemDatabase)
	databaseRepository := repositories.NewDatabaseRepository(
		systemDatabase,
		infrastructure.credentialCipher,
	)
	sessionRepository := repositories.NewSessionRepository(systemDatabase)
	migrationRepository := repositories.NewMigrationRepository(systemDatabase)
	draftRepository := repositories.NewDraftRepository(systemDatabase)

	sessionService := sessionservice.NewSessionService(sessionRepository)
	authService := authservice.NewAuthService(userRepository, sessionService)
	userService := userservice.NewUserService(userRepository, groupRepository, sessionService, userGroupRepository)
	userGroupService := usergroupservice.NewUserGroupService(userGroupRepository, groupRepository, userRepository, sessionService)
	groupService := groupservice.NewGroupService(
		userGroupRepository,
		groupRepository,
		userRepository,
		databaseRepository,
		sessionService,
	)
	databaseService := databaseservice.NewDatabaseService(
		databaseRepository,
		infrastructure.connectionTester,
		userGroupRepository,
		sessionService,
		infrastructure.poolManager,
	)
	databaseDriverService := databasedriverservice.NewDatabaseDriverService(
		databaseService,
		databaseRepository,
		infrastructure.databaseDriver,
		userGroupService,
	)
	migrationService := migrationservice.NewMigrationService(
		migrationRepository,
		databaseRepository,
		infrastructure.databaseDriver,
		userGroupService,
	)

	migrationMetrics := metrics.NewPrometheusMigrationMetrics()
	worker := draftservice.NewExecutionWorker(
		draftRepository,
		databaseRepository,
		migrationRepository,
		infrastructure.databaseDriver,
		groupRepository,
		nil,
		migrationMetrics,
		userGroupService,
	)

	taskQueue := queuepostgres.NewQueue(
		systemDatabase,
		worker.Execute,
		workerConfig.Count,
		workerConfig.QueueCapacity,
		channel.DefaultTaskTimeout,
	)

	draftService := draftservice.NewDraftService(
		draftRepository,
		groupRepository,
		userGroupService,
		databaseRepository,
		taskQueue,
		migrationRepository,
		infrastructure.databaseDriver,
	)

	return &applicationServices{
		auth:           authService,
		user:           userService,
		group:          groupService,
		userGroup:      userGroupService,
		session:        sessionService,
		database:       databaseService,
		databaseDriver: databaseDriverService,
		migration:      migrationService,
		draft:          draftService,
		queue:          taskQueue,
	}, nil
}
