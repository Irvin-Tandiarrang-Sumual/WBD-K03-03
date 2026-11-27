import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { databaseService } from '../services/database_service';
import { toggleColumnVisibility } from '../utils/table';
import type { DbEngine, DbStatus, TargetDatabase } from '../types';
import { useAsyncAction } from './useAsyncAction';

interface DatabaseFilters {
  search: string;
  engine: 'all' | DbEngine;
  status: 'all' | DbStatus;
}

interface SaveDbInput {
  name: string;
  engine: DbEngine;
  host: string;
  port: string;
}

const DEFAULT_COLUMNS: Record<string, boolean> = {
  engine: true,
  hostPort: true,
  groups: true,
  status: true,
};

export function useAdminDatabases(initialDatabases: TargetDatabase[]) {
  const { showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [databases, setDatabases] = useState<TargetDatabase[]>(initialDatabases);
  const [filters, setFilters] = useState<DatabaseFilters>({
    search: '',
    engine: 'all',
    status: 'all',
  });

  const [columnsMap, setColumnsMap] = useState<Record<string, boolean>>(DEFAULT_COLUMNS);
  const [registerModalOpen, setRegisterModalOpen] = useState(false);
  const [editingDb, setEditingDb] = useState<TargetDatabase | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  const filteredDatabases = useMemo(
    () => filterDatabasesList(databases, filters),
    [databases, filters]
  );

  const handleTestConnection = async (db: TargetDatabase) => {
    setTestingId(db.id);
    try {
      await runAsync(
        async () => {
          const res = await databaseService.testConnection(db.id);
          return res;
        },
        {
          successMessage: (res) =>
            res.success
              ? `Connected to ${db.name} (${db.hostPort}) · ${res.latencyMs}ms`
              : `Connection to ${db.name} degraded (${res.latencyMs}ms)`,
          successTone: 'pine',
          errorMessage: `Failed to test connection to ${db.name}`,
        }
      );
    } finally {
      setTestingId(null);
    }
  };

  const handleSaveDb = async (data: SaveDbInput) => {
    await runAsync(
      async () => {
        const portNumber = parseInt(data.port, 10) || (data.engine === 'PostgreSQL' ? 5432 : 3306);
        const saved = await databaseService.createAdminDatabase({
          name: data.name,
          type: data.engine === 'PostgreSQL' ? 'postgresql' : 'mysql',
          host: data.host,
          port: portNumber,
          user: data.engine === 'PostgreSQL' ? 'postgres' : 'root',
          password: 'password',
        });
        return saved;
      },
      {
        successMessage: (saved) => `Saved database ${saved.name}`,
        successTone: 'pine',
        onSuccess: (saved) => {
          setDatabases((prev) => upsertDatabase(prev, saved));
          setRegisterModalOpen(false);
          setEditingDb(null);
        },
      }
    );
  };

  const toggleColumn = (key: string) => {
    setColumnsMap((prev) => toggleColumnVisibility(prev, key, 1));
  };

  return {
    databases,
    filteredDatabases,
    search: filters.search,
    setSearch: (search: string) => setFilters((prev) => ({ ...prev, search })),
    engineFilter: filters.engine,
    setEngineFilter: (engine: 'all' | DbEngine) => setFilters((prev) => ({ ...prev, engine })),
    statusFilter: filters.status,
    setStatusFilter: (status: 'all' | DbStatus) => setFilters((prev) => ({ ...prev, status })),
    columnsMap,
    toggleColumn,
    registerModalOpen,
    setRegisterModalOpen,
    editingDb,
    setEditingDb,
    testingId,
    handleTestConnection,
    handleSaveDb,
  };
}

// --- Pure Helper Functions (Stepdown Rule) ---

function filterDatabasesList(databases: TargetDatabase[], filters: DatabaseFilters): TargetDatabase[] {
  const query = filters.search.trim().toLowerCase();

  return databases.filter((db) => {
    if (query && !db.name.toLowerCase().includes(query)) return false;
    if (filters.engine !== 'all' && db.engine !== filters.engine) return false;
    if (filters.status !== 'all' && db.status !== filters.status) return false;
    return true;
  });
}

function upsertDatabase(list: TargetDatabase[], item: TargetDatabase): TargetDatabase[] {
  const existingIndex = list.findIndex((db) => db.id === item.id);
  if (existingIndex >= 0) {
    const updated = [...list];
    updated[existingIndex] = item;
    return updated;
  }
  return [item, ...list];
}
