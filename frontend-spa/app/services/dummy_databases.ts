import type { TargetDatabase, DbEngine, DbStatus } from '~/types/database';

export const INITIAL_DATABASES: TargetDatabase[] = [
  {
    id: 'db-payments',
    name: 'payments_prod_db',
    engine: 'PostgreSQL',
    driverVersion: 'PostgreSQL 16.4',
    host: 'pg-payments-01.internal',
    port: 5432,
    hostPort: 'pg-payments-01.internal:5432',
    status: 'Degraded',
    quarantined: true,
    quarantineDraftId: 'DRF-204',
    headVersion: 4,
    redoStack: [],
  },
  {
    id: 'db-ledger',
    name: 'core_ledger_pg',
    engine: 'PostgreSQL',
    driverVersion: 'PostgreSQL 16.4',
    host: 'pg-ledger-02.internal',
    port: 5432,
    hostPort: 'pg-ledger-02.internal:5432',
    status: 'Healthy',
    quarantined: false,
    headVersion: 3,
    redoStack: [],
  },
  {
    id: 'db-analytics',
    name: 'analytics_warehouse_mysql',
    engine: 'MySQL',
    driverVersion: 'MySQL 8.0.39',
    host: 'mysql-bi-01.internal',
    port: 3306,
    hostPort: 'mysql-bi-01.internal:3306',
    status: 'Healthy',
    quarantined: false,
    headVersion: 2,
    redoStack: [],
  },
  {
    id: 'db-auth',
    name: 'auth_identity_pg',
    engine: 'PostgreSQL',
    driverVersion: 'PostgreSQL 15.8',
    host: 'pg-auth-01.internal',
    port: 5432,
    hostPort: 'pg-auth-01.internal:5432',
    status: 'Healthy',
    quarantined: false,
    headVersion: 5,
    redoStack: [],
  },
];

let memoryDatabases: TargetDatabase[] = [...INITIAL_DATABASES];

export const fetchDatabases = async (): Promise<TargetDatabase[]> => {
  await new Promise((resolve) => setTimeout(resolve, 80));
  return [...memoryDatabases];
};

export const fetchDatabaseById = async (id: string): Promise<TargetDatabase | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 50));
  return memoryDatabases.find((db) => db.id === id);
};

export const saveDatabase = async (input: {
  id?: string;
  name: string;
  engine: DbEngine;
  host?: string;
  port?: number | string;
  hostPort?: string;
  status?: DbStatus;
}): Promise<TargetDatabase> => {
  await new Promise((resolve) => setTimeout(resolve, 100));
  let host = input.host;
  let port = typeof input.port === 'string' ? parseInt(input.port, 10) : input.port;
  if (!host && input.hostPort) {
    const parts = input.hostPort.split(':');
    host = parts[0];
    port = parseInt(parts[1] || '5432', 10);
  }
  host = host || 'localhost';
  port = port || 5432;

  if (input.id) {
    const existing = memoryDatabases.find((d) => d.id === input.id);
    if (!existing) throw new Error('Database not found');
    const updated: TargetDatabase = {
      ...existing,
      name: input.name,
      engine: input.engine,
      host,
      port,
      hostPort: `${host}:${port}`,
      status: input.status || existing.status,
    };
    memoryDatabases = memoryDatabases.map((d) => (d.id === input.id ? updated : d));
    return updated;
  }

  const newDb: TargetDatabase = {
    id: `db-${input.name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`,
    name: input.name,
    engine: input.engine,
    driverVersion: input.engine === 'PostgreSQL' ? 'PostgreSQL 16.4' : 'MySQL 8.0.39',
    host,
    port,
    hostPort: `${host}:${port}`,
    status: input.status || 'Healthy',
    quarantined: false,
    headVersion: 0,
    redoStack: [],
  };
  memoryDatabases = [newDb, ...memoryDatabases];
  return newDb;
};

export const testDatabaseConnection = async (id: string): Promise<{ success: boolean; latencyMs: number }> => {
  await new Promise((resolve) => setTimeout(resolve, 300));
  const db = memoryDatabases.find((d) => d.id === id);
  if (!db) throw new Error('Database not found');
  const latency = Math.floor(Math.random() * 25) + 12;
  return { success: db.status === 'Healthy', latencyMs: latency };
};

export const updateDatabaseQuarantine = (dbId: string, quarantined: boolean, draftId?: string) => {
  memoryDatabases = memoryDatabases.map((db) => {
    if (db.id !== dbId) return db;
    return {
      ...db,
      quarantined,
      quarantineDraftId: quarantined ? draftId : undefined,
      status: quarantined ? 'Degraded' : 'Healthy',
    };
  });
};

export const updateDatabaseHeadVersion = (dbId: string, nextHead: number, redoStack: number[]) => {
  memoryDatabases = memoryDatabases.map((db) => {
    if (db.id !== dbId) return db;
    return { ...db, headVersion: nextHead, redoStack };
  });
};
