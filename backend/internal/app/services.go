package app

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	databaseexecutorservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_executor"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	usergroupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user_group"
	"gorm.io/gorm"
)

type applicationServices struct {
	auth             ports.AuthService
	user             ports.UserService
	group            ports.GroupService
	userGroup        ports.UserGroupService
	session          ports.SessionService
	database         ports.DatabaseService
	databaseExecutor ports.DatabaseExecutorService
	migration        ports.MigrationService
}

func buildServices(
	systemDatabase *gorm.DB,
	infrastructure *targetDatabaseInfrastructure,
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
	databaseExecutorService := databaseexecutorservice.NewDatabaseExecutorService(
		databaseService,
		databaseRepository,
		infrastructure.databaseExecutor,
		userGroupService,
	)
	migrationService := migrationservice.NewMigrationService(
		migrationRepository,
		databaseRepository,
		infrastructure.databaseExecutor,
		userGroupService,
	)

	return &applicationServices{
		auth:             authService,
		user:             userService,
		group:            groupService,
		userGroup:        userGroupService,
		session:          sessionService,
		database:         databaseService,
		databaseExecutor: databaseExecutorService,
		migration:        migrationService,
	}, nil
}
