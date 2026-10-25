package app

import (
	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	migrationrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/migration"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	databaseexecutorservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_executor"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/gorm"
)

type applicationServices struct {
	auth             ports.AuthService
	user             ports.UserService
	group            ports.GroupService
	session          ports.SessionService
	database         ports.DatabaseService
	databaseExecutor ports.DatabaseExecutorService
	migration        ports.MigrationService
}

func buildServices(
	systemDatabase *gorm.DB,
	infrastructure *targetDatabaseInfrastructure,
) (*applicationServices, error) {
	userRepository := userrepository.NewUserRepository(systemDatabase)
	groupRepository := grouprepository.NewGroupRepository(systemDatabase)
	databaseRepository := databaserepository.NewDatabaseRepository(
		systemDatabase,
		infrastructure.credentialCipher,
	)
	sessionRepository := sessionrepository.NewSessionRepository(systemDatabase)
	migrationRepository := migrationrepository.NewMigrationRepository(systemDatabase)

	sessionService := sessionservice.NewSessionService(sessionRepository)
	authService := authservice.NewAuthService(userRepository, sessionService)
	userService := userservice.NewUserService(userRepository, groupRepository, sessionService)
	groupService := groupservice.NewGroupService(
		groupRepository,
		userRepository,
		databaseRepository,
		sessionService,
	)
	databaseService := databaseservice.NewDatabaseService(
		databaseRepository,
		infrastructure.connectionTester,
		groupRepository,
		sessionService,
		infrastructure.poolManager,
	)
	databaseExecutorService := databaseexecutorservice.NewDatabaseExecutorService(
		databaseService,
		databaseRepository,
		infrastructure.databaseExecutor,
		groupService,
	)
	migrationService := migrationservice.NewMigrationService(
		migrationRepository,
		databaseRepository,
		infrastructure.databaseExecutor,
		groupService,
	)

	return &applicationServices{
		auth:             authService,
		user:             userService,
		group:            groupService,
		session:          sessionService,
		database:         databaseService,
		databaseExecutor: databaseExecutorService,
		migration:        migrationService,
	}, nil
}
