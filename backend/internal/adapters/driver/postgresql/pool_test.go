package postgresql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestPoolManagerReturnsExistingPool(t *testing.T) {
	manager := NewPoolManager()

	pool, err := sql.Open("pgx", "postgres://localhost/test")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	defer pool.Close()

	manager.pools[1] = pool

	database := domain.Database{ID: 1}

	got, err := manager.Get(context.Background(), database)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != pool {
		t.Fatal("Get() did not return existing pool")
	}
}

func TestPoolManagerInvalidate(t *testing.T) {
	manager := NewPoolManager()

	pool, err := sql.Open("pgx", "postgres://localhost/test")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

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

	first, err := sql.Open("pgx", "postgres://localhost/first")
	if err != nil {
		t.Fatalf("first sql.Open() error = %v", err)
	}

	second, err := sql.Open("pgx", "postgres://localhost/second")
	if err != nil {
		t.Fatalf("second sql.Open() error = %v", err)
	}

	manager.pools[1] = first
	manager.pools[2] = second

	if err := manager.CloseAll(); err != nil {
		t.Fatalf("CloseAll() error = %v", err)
	}

	if len(manager.pools) != 0 {
		t.Fatalf("pool count = %d, want 0", len(manager.pools))
	}
}