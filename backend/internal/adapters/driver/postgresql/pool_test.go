//go:build integration

package postgresql

import (
	"context"
	"os"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openPoolTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("PLATFORM_DATABASE_URL")
	if dsn == "" {
		t.Skip("PLATFORM_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	pool, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	return pool
}

func TestPoolManagerReturnsExistingPool(t *testing.T) {
	manager := NewPoolManager()
	pool := openPoolTestDatabase(t)
	sqlPool, err := pool.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	defer sqlPool.Close()

	manager.pools[1] = pool
	got, err := manager.Get(context.Background(), domain.Database{ID: 1})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != pool {
		t.Fatal("Get() did not return existing pool")
	}
}

func TestPoolManagerInvalidate(t *testing.T) {
	manager := NewPoolManager()
	pool := openPoolTestDatabase(t)
	manager.pools[1] = pool

	if err := manager.Invalidate(1); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}
	if _, exists := manager.pools[1]; exists {
		t.Fatal("Invalidate() did not remove pool")
	}
	if err := manager.Invalidate(1); err != nil {
		t.Fatalf("second Invalidate() error = %v", err)
	}
}

func TestPoolManagerCloseAll(t *testing.T) {
	manager := NewPoolManager()
	manager.pools[1] = openPoolTestDatabase(t)
	manager.pools[2] = openPoolTestDatabase(t)

	if err := manager.CloseAll(); err != nil {
		t.Fatalf("CloseAll() error = %v", err)
	}
	if len(manager.pools) != 0 {
		t.Fatalf("pool count = %d, want 0", len(manager.pools))
	}
}
