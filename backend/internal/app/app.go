package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	httpmiddleware "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/logger"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/telemetry"
	"github.com/gin-gonic/gin"

	_ "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/docs"
)

func Run() {
	logOutput, closeLog, err := applicationLogOutput(os.Getenv("APP_LOG_PATH"))
	slog.SetDefault(logger.NewJSONLogger(logOutput, slog.LevelInfo))
	if err != nil {
		slog.Warn("failed to open application log file", "event", "logging.file.unavailable")
	}
	defer closeLog()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err.Error())
		os.Exit(1)
	}

	shutdownTracer, err := telemetry.InitTracer(context.Background(), telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Exporter:    cfg.Telemetry.Exporter,
		Endpoint:    cfg.Telemetry.Endpoint,
	})
	if err != nil {
		slog.Warn("failed to initialize tracer provider", "error", err.Error())
	} else if shutdownTracer != nil {
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				slog.Error("failed to shutdown tracer provider", "error", err.Error())
			}
		}()
	}

	application, err := buildApplication(cfg)
	if err != nil {
		slog.Error("failed to build application", "error", err.Error())
		os.Exit(1)
	}
	defer application.close()

	address := fmt.Sprintf(
		"%s:%d",
		application.config.Server.Host,
		application.config.Server.Port,
	)

	slog.Info("starting HTTP server", "address", address)
	if err := application.engine.Run(address); err != nil {
		slog.Error("failed to run HTTP server", "error", err.Error())
		os.Exit(1)
	}
}

func applicationLogOutput(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return os.Stdout, func() {}, err
	}
	return io.MultiWriter(os.Stdout, file), func() { _ = file.Close() }, nil
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

func buildApplication(configuration *config.Config) (*application, error) {

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

	handlers := buildHandlers(configuration, services, systemDatabase, targetDatabaseInfrastructure.poolManager)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httpmiddleware.CORS(configuration.Server.FrontendOrigin))
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"swagger": "/swagger/index.html",
			"health":  "/health",
			"metrics": "/metrics",
		})
	})
	engine.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	engine.GET("/health", handlers.health.Check)
	engine.GET("/metrics", handlers.metrics.GetMetrics)
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
