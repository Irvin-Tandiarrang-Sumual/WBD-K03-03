//go:build integration

package integration_test

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	driver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	migrationrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/migration"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	databaseexecutorservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database_executor"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	migrationservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"
	"gorm.io/gorm"
)

const integrationCredentialKey = "0123456789abcdef0123456789abcdef"

type integrationStack struct {
	db               *gorm.DB
	target           domain.Database
	user             domain.User
	group            domain.Group
	databaseRepo     ports.DatabaseRepository
	groupRepo        ports.GroupRepository
	userRepo         ports.UserRepository
	sessionRepo      ports.SessionRepository
	migrationRepo    ports.MigrationRepository
	sessionService   ports.SessionService
	groupService     ports.GroupService
	userService      ports.UserService
	databaseService  ports.DatabaseService
	executor         ports.DatabaseExecutor
	executorService  ports.DatabaseExecutorService
	migrationService ports.MigrationService
	poolManager      *driver.DatabasePoolManager
}

func targetDatabaseFromPlatformDSN(t *testing.T) domain.Database {
	t.Helper()
	dsn := os.Getenv("PLATFORM_DATABASE_URL")
	if dsn == "" {
		t.Skip("PLATFORM_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PLATFORM_DATABASE_URL: %v", err)
	}
	port := uint(5432)
	if parsed.Port() != "" {
		value, err := strconv.ParseUint(parsed.Port(), 10, 16)
		if err != nil {
			t.Fatalf("invalid platform database port: %v", err)
		}
		port = uint(value)
	}
	password, _ := parsed.User.Password()
	return domain.Database{
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		Type:     "postgresql",
		Name:     strings.TrimPrefix(parsed.Path, "/"),
	}
}

func newIntegrationStack(t *testing.T, access domain.AccessType) *integrationStack {
	t.Helper()
	db := openTestDB(t)
	ctx := context.Background()

	cipher, err := security.NewAESGCMCipher([]byte(integrationCredentialKey))
	if err != nil {
		t.Fatalf("create credential cipher: %v", err)
	}

	databaseRepo := databaserepository.NewDatabaseRepository(db, cipher)
	groupRepo := grouprepository.NewGroupRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	sessionRepo := sessionrepository.NewSessionRepository(db)
	migrationRepo := migrationrepository.NewMigrationRepository(db)
	sessionService := sessionservice.NewSessionService(sessionRepo)
	groupService := groupservice.NewGroupService(groupRepo, userRepo, databaseRepo, sessionService)
	userService := userservice.NewUserService(userRepo, groupRepo, sessionService)

	targetInput := targetDatabaseFromPlatformDSN(t)
	createdTarget, err := databaseRepo.CreateDatabase(ctx, &targetInput)
	if err != nil {
		t.Fatalf("register integration target database: %v", err)
	}
	targetInput.ID = createdTarget.ID

	group, err := groupService.Create(ctx, domain.CreateGroupInput{
		Name: "integration-target-users",
		DatabaseAccesses: []domain.DatabaseAccess{{
			DatabaseID: targetInput.ID,
			AccessType: access,
		}},
	})
	if err != nil {
		t.Fatalf("create integration group: %v", err)
	}

	user, err := userService.Create(ctx, domain.CreateUserInput{
		Username: "integration-target-user",
		Password: "secret123",
		GroupIDs: []uint{group.ID},
	})
	if err != nil {
		t.Fatalf("create integration user: %v", err)
	}

	poolManager := driver.NewDatabasePoolManager()
	t.Cleanup(func() {
		if err := poolManager.CloseAll(); err != nil {
			t.Errorf("close target database pools: %v", err)
		}
	})
	executor := driver.NewDatabaseExecutor(poolManager)
	databaseService := databaseservice.NewDatabaseService(databaseRepo, nil, groupRepo, sessionService, poolManager)
	executorService := databaseexecutorservice.NewDatabaseExecutorService(databaseService, databaseRepo, executor, groupService)
	migrationService := migrationservice.NewMigrationService(migrationRepo, databaseRepo, executor, groupService)

	return &integrationStack{
		db:               db,
		target:           targetInput,
		user:             user,
		group:            group,
		databaseRepo:     databaseRepo,
		groupRepo:        groupRepo,
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		migrationRepo:    migrationRepo,
		sessionService:   sessionService,
		groupService:     groupService,
		userService:      userService,
		databaseService:  databaseService,
		executor:         executor,
		executorService:  executorService,
		migrationService: migrationService,
		poolManager:      poolManager,
	}
}
