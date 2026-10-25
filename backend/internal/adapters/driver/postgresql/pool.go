package postgresql

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PoolManager struct {
	mu    sync.RWMutex
	pools map[uint]*gorm.DB
}

var _ ports.DatabasePoolInvalidator = (*PoolManager)(nil)

func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[uint]*gorm.DB),
	}
}

func (m *PoolManager) Get(ctx context.Context, database domain.Database) (*gorm.DB, error) {
	m.mu.RLock()
	existingPool, exists := m.pools[database.ID]
	m.mu.RUnlock()
	if exists {
		return existingPool, nil
	}

	dsn := buildDSN(database)

	pool, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}

	sqlPool, err := pool.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL SQL pool: %w", err)
	}
	if err := sqlPool.PingContext(ctx); err != nil {
		_ = sqlPool.Close()
		return nil, fmt.Errorf("ping PostgreSQL pool: %w", err)
	}

	m.mu.Lock()

	if existingPool, exists := m.pools[database.ID]; exists {
		m.mu.Unlock()

		if sqlPool, err := pool.DB(); err == nil {
			_ = sqlPool.Close()
		}

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

	sqlPool, err := pool.DB()
	if err != nil {
		return err
	}
	return sqlPool.Close()
}

func (m *PoolManager) CloseAll() error {
	m.mu.Lock()

	pools := make([]*gorm.DB, 0, len(m.pools))

	for databaseID, pool := range m.pools {
		pools = append(pools, pool)
		delete(m.pools, databaseID)
	}

	m.mu.Unlock()

	var closeErrors []error

	for _, pool := range pools {
		sqlPool, err := pool.DB()
		if err != nil {
			closeErrors = append(closeErrors, err)
			continue
		}
		if err := sqlPool.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}

	return errors.Join(closeErrors...)
}