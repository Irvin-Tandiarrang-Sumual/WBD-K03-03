package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	httpmiddleware "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/gin-gonic/gin"
)

func Run() {
	application, err := buildApplication()
	if err != nil {
		log.Fatalf("failed to build application: %v", err)
	}
	defer application.close()

	address := fmt.Sprintf(
		"%s:%d",
		application.config.Server.Host,
		application.config.Server.Port,
	)

	if err := application.engine.Run(address); err != nil {
		log.Fatalf("failed to run HTTP server: %v", err)
	}
}

type application struct {
	config      *config.Config
	engine      *gin.Engine
	closeTarget func()
}

func (a *application) close() {
	if a.closeTarget != nil {
		a.closeTarget()
	}
}

func buildApplication() (*application, error) {
	configuration, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	systemDatabase, closeSystemDatabase, err := buildSystemDatabase(configuration)
	if err != nil {
		return nil, err
	}

	targetDatabaseInfrastructure, err := buildTargetDatabaseInfrastructure(configuration)
	if err != nil {
		closeSystemDatabase()
		return nil, err
	}

	services, err := buildServices(systemDatabase, targetDatabaseInfrastructure)
	if err != nil {
		targetDatabaseInfrastructure.close()
		closeSystemDatabase()
		return nil, err
	}

	handlers := buildHandlers(configuration, services)
	engine := gin.Default()
	engine.Use(httpmiddleware.CORS(configuration.Server.FrontendOrigin))
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.SetupRouter(
		engine,
		handlers.auth,
		handlers.user,
		handlers.group,
		handlers.database,
		handlers.databaseExecutor,
		handlers.migration,
		services.session,
		services.user,
		services.group,
	)

	return &application{
		config: configuration,
		engine: engine,
		closeTarget: func() {
			targetDatabaseInfrastructure.close()
			closeSystemDatabase()
		},
	}, nil
}
