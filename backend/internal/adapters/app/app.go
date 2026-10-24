package app

import (
	"fmt"
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"

	postgresqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/postgresql"

	authhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/auth"
	databasehandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/database"
	grouphandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/group"
	userhandler "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/handlers/user"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"

	storagepostgresql "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql"

	databaserepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/database"
	grouprepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/group"
	sessionrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/session"
	userrepository "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/repositories/user"

	authservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/auth"
	databaseservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/database"
	groupservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/group"
	sessionservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/session"
	userservice "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/user"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations"
	"github.com/gin-gonic/gin"
)

func Run() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("failed to load config: %v", err)
		return
	}

	db, err := storagepostgresql.ConnectToSystemDatabase(
		cfg.Database,
	)

	if err != nil {
		log.Printf(
			"failed to connect to system database: %v",
			err,
		)
		return
	}

	if err := migrations.Migrate(db); err != nil {
		log.Printf(
			"failed to migrate system database: %v",
			err,
		)
		return
	}

	credentialCipher, err := security.NewAESGCMCipher(
		[]byte(cfg.Security.DatabaseCredentialKey),
	)

	if err != nil {
		log.Printf(
			"failed to initialize database credential cipher: %v",
			err,
		)
		return
	}

	connectionTester := postgresqldriver.NewConnectionTester()

	poolManager := postgresqldriver.NewPoolManager()

	defer func() {
		if err := poolManager.CloseAll(); err != nil {
			log.Printf(
				"failed to close target database pools: %v",
				err,
			)
		}
	}()

	userRepository := userrepository.NewRepository(
		db,
	)

	groupRepository := grouprepository.NewRepository(
		db,
	)

	databaseRepository := databaserepository.NewRepository(
		db,
		credentialCipher,
	)

	sessionRepository := sessionrepository.NewRepository(
		db,
	)

	sessionService := sessionservice.NewService(
		sessionRepository,
	)

	authService := authservice.NewService(
		userRepository,
		sessionService,
	)

	userService := userservice.NewService(
		userRepository,
		groupRepository,
		sessionService,
	)

	groupService := groupservice.NewService(
		groupRepository,
		userRepository,
		databaseRepository,
		sessionService,
	)

	databaseService := databaseservice.NewService(
		databaseRepository,
		connectionTester,
		groupRepository,
		sessionService,
		poolManager,
	)

	authHandler := authhandler.NewHandler(
		authService,
		cfg.Session.CookieSecure,
	)

	userHandler := userhandler.NewHandler(
		userService,
	)

	groupHandler := grouphandler.NewHandler(
		groupService,
	)

	databaseHandler := databasehandler.NewHandler(
		databaseService,
	)

	r := gin.Default()

	router.SetUpRouter(
		r,
		authHandler,
		userHandler,
		groupHandler,
		databaseHandler,
		sessionService,
		userService,
	)

	address := fmt.Sprintf(
		"%s:%d",
		cfg.Server.Host,
		cfg.Server.Port,
	)

	if err := r.Run(address); err != nil {
		log.Printf(
			"failed to run HTTP server: %v",
			err,
		)
	}
}