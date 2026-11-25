package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	httpmiddleware "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/gin-gonic/gin"

	_ "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/docs"
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

	services, err := buildServices(systemDatabase, targetDatabaseInfrastructure, configuration.Worker)
	if err != nil {
		targetDatabaseInfrastructure.close()
		closeSystemDatabase()
		return nil, err
	}

	middlewares := buildMiddlewares(
		services.session,
		services.user,
		configuration.Session.CookieSecure,
		configuration.Server.FrontendOrigin,
	)

	handlers := buildHandlers(configuration, services, systemDatabase)
	engine := gin.Default()
	engine.Use(httpmiddleware.CORS(configuration.Server.FrontendOrigin))
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"swagger": "/swagger/index.html",
			"health":  "/health",
		})
	})
	engine.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	engine.GET("/health", handlers.health.Check)
	router.SetupRouter(
		engine,
		middlewares.authMiddleware,
		handlers.auth,
		handlers.user,
		handlers.group,
		handlers.userGroup,
		handlers.database,
		handlers.databaseDriver,
		handlers.migration,
		handlers.draft,
		handlers.health,
		services.session,
		services.user,
		middlewares.csrfMiddleware,
	)

	return &application{
		config: configuration,
		engine: engine,
		closeTarget: func() {
			services.close()
			targetDatabaseInfrastructure.close()
			closeSystemDatabase()
		},
	}, nil
}
