package app

import (
	"fmt"
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/gin-gonic/gin"
)

func Run() {
	application, err := buildApplication()
	if err != nil {
		log.Printf("failed to build application: %v", err)
		return
	}
	defer application.close()

	address := fmt.Sprintf(
		"%s:%d",
		application.config.Server.Host,
		application.config.Server.Port,
	)

	if err := application.engine.Run(address); err != nil {
		log.Printf("failed to run HTTP server: %v", err)
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
