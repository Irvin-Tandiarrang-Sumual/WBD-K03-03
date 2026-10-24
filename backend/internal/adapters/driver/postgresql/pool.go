package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type PoolManager struct {
	mu    sync.RWMutex
	pools map[uint]*sql.DB
}

var _ ports.DatabasePoolInvalidator = (*PoolManager)(nil)

func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[uint]*sql.DB),
	}
}

func (m *PoolManager) Get(ctx context.Context, database domain.Database) (*sql.DB, error) {
	m.mu.RLock()
	existingPool, exists := m.pools[database.ID]
	m.mu.RUnlock()
	if exists {
		return existingPool, nil
	}

	dsn := buildDSN(database)

	pool, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}

	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL pool: %w", err)
	}

	m.mu.Lock()

	if existingPool, exists := m.pools[database.ID]; exists {
		m.mu.Unlock()

		pool.Close()

		return existingPool, nil
	}

	m.pools[database.ID] = pool
	m.mu.Unlock()

	return pool, nil
}

func (m *PoolManager) Invalidate(databaseID uint) error {
	
	m.mu.Lock()

	pool, exists := m.pools[databaseID]

	if exists {
		delete(m.pools, databaseID)
	}

	m.mu.Unlock()

	if !exists {
		return nil
	}

	return pool.Close()
}

func (m *PoolManager) CloseAll() error {
	m.mu.Lock()

	pools := make([]*sql.DB, 0, len(m.pools))

	for databaseID, pool := range m.pools {
		pools = append(pools, pool)
		delete(m.pools, databaseID)
	}

	m.mu.Unlock()

	var closeErrors []error

	for _, pool := range pools {
		if err := pool.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}

	return errors.Join(closeErrors...)
}