//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/migrations"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	tc "github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	testcontainersReady bool

	platformGormDB *gorm.DB

	targetPostgresConfig domain.Database
	targetMySQLConfig    domain.Database

	platformContainer       *tcpostgres.PostgresContainer
	targetPostgresContainer *tcpostgres.PostgresContainer
	targetMySQLContainer    *tcmysql.MySQLContainer
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Try spinning up Testcontainers if Docker is available
	requireContainers := os.Getenv("REQUIRE_INTEGRATION_CONTAINERS") == "true"
	err := setupTestcontainers(ctx)
	if err != nil {
		log.Printf("[TestMain] Testcontainers setup skipped/failed: %v", err)
		if !requireContainers {
			log.Println("[TestMain] Attempting fallback to environment variable DSNs...")
			setupEnvironmentDSNs()
		}
	}

	code := 1
	if err != nil && requireContainers {
		log.Printf("[TestMain] Required testcontainers could not start: %v", err)
	} else {
		code = m.Run()
	}

	// 2. Teardown Testcontainers
	teardownCtx, teardownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer teardownCancel()
	teardownTestcontainers(teardownCtx)

	os.Exit(code)
}

func setupTestcontainers(ctx context.Context) error {
	log.Println("[TestMain] Starting 3 containers in parallel (Platform PostgreSQL, Target PostgreSQL, Target MySQL)...")

	var (
		wg          sync.WaitGroup
		errOnce     sync.Once
		parallelErr error
	)

	setErr := func(err error) {
		if err != nil {
			errOnce.Do(func() {
				parallelErr = err
			})
		}
	}

	wg.Add(3)

	// 1. Platform PostgreSQL
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				setErr(fmt.Errorf("panic starting platform postgres: %v", r))
			}
		}()
		c, err := tcpostgres.Run(ctx,
			"postgres:15-alpine",
			tcpostgres.WithDatabase("platform_integration_test"),
			tcpostgres.WithUsername("platform_test_user"),
			tcpostgres.WithPassword("platform_test_pass"),
			tcpostgres.BasicWaitStrategies(),
		)
		if err != nil {
			setErr(fmt.Errorf("start platform postgres container: %w", err))
			return
		}
		platformContainer = c

		platformConnStr, err := platformContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setErr(fmt.Errorf("get platform conn string: %w", err))
			return
		}

		db, err := gorm.Open(gormpostgres.Open(platformConnStr), &gorm.Config{
			TranslateError: true,
		})
		if err != nil {
			setErr(fmt.Errorf("open platform gorm DB: %w", err))
			return
		}

		if err := migrations.Migrate(db); err != nil {
			setErr(fmt.Errorf("migrate platform DB: %w", err))
			return
		}
		platformGormDB = db
	}()

	// 2. Target PostgreSQL
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				setErr(fmt.Errorf("panic starting target postgres: %v", r))
			}
		}()
		c, err := tcpostgres.Run(ctx,
			"postgres:15-alpine",
			tcpostgres.WithDatabase("target_pg_integration_test"),
			tcpostgres.WithUsername("target_pg_user"),
			tcpostgres.WithPassword("target_pg_pass"),
			tcpostgres.BasicWaitStrategies(),
		)
		if err != nil {
			setErr(fmt.Errorf("start target postgres container: %w", err))
			return
		}
		targetPostgresContainer = c

		pgHost, err := targetPostgresContainer.Host(ctx)
		if err != nil {
			setErr(fmt.Errorf("get target postgres host: %w", err))
			return
		}
		pgPort, err := targetPostgresContainer.MappedPort(ctx, "5432")
		if err != nil {
			setErr(fmt.Errorf("get target postgres port: %w", err))
			return
		}
		pgPortInt, _ := strconv.ParseUint(pgPort.Port(), 10, 32)

		targetPostgresConfig = domain.Database{
			Host:     pgHost,
			Port:     uint(pgPortInt),
			User:     "target_pg_user",
			Password: "target_pg_pass",
			Type:     "postgresql",
			Name:     "target_pg_integration_test",
		}
	}()

	// 3. Target MySQL
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				setErr(fmt.Errorf("panic starting target mysql: %v", r))
			}
		}()
		c, err := tcmysql.Run(ctx,
			"mysql:8.4",
			tcmysql.WithDatabase("target_mysql_integration_test"),
			tcmysql.WithUsername("target_my_user"),
			tcmysql.WithPassword("target_my_pass"),
		)
		if err != nil {
			setErr(fmt.Errorf("start target mysql container: %w", err))
			return
		}
		targetMySQLContainer = c

		myHost, err := targetMySQLContainer.Host(ctx)
		if err != nil {
			setErr(fmt.Errorf("get target mysql host: %w", err))
			return
		}
		myPort, err := targetMySQLContainer.MappedPort(ctx, "3306")
		if err != nil {
			setErr(fmt.Errorf("get target mysql port: %w", err))
			return
		}
		myPortInt, _ := strconv.ParseUint(myPort.Port(), 10, 32)

		targetMySQLConfig = domain.Database{
			Host:     myHost,
			Port:     uint(myPortInt),
			User:     "target_my_user",
			Password: "target_my_pass",
			Type:     "mysql",
			Name:     "target_mysql_integration_test",
		}
	}()

	wg.Wait()

	if parallelErr != nil {
		return parallelErr
	}

	testcontainersReady = true
	log.Println("[TestMain] All 3 containers successfully started in parallel and ready!")
	return nil
}

func setupEnvironmentDSNs() {
	platformDSN := os.Getenv("PLATFORM_DATABASE_URL")
	if platformDSN == "" {
		platformDSN = os.Getenv("TEST_PLATFORM_POSTGRES_DSN")
	}

	if platformDSN != "" {
		var err error
		platformGormDB, err = gorm.Open(gormpostgres.Open(platformDSN), &gorm.Config{
			TranslateError: true,
		})
		if err == nil {
			_ = migrations.Migrate(platformGormDB)
			testcontainersReady = true
			log.Println("[TestMain] Connected to fallback Platform PostgreSQL via environment DSN")
		}
	}
}

func teardownTestcontainers(ctx context.Context) {
	var wg sync.WaitGroup
	terminate := func(c tc.Container) {
		defer wg.Done()
		if c != nil {
			_ = tc.TerminateContainer(c)
		}
	}

	wg.Add(3)
	go terminate(platformContainer)
	go terminate(targetPostgresContainer)
	go terminate(targetMySQLContainer)
	wg.Wait()
}
