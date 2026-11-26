import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { saveDatabase, testDatabaseConnection } from '../services/dummy_databases';
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
    await runAsync(
      async () => {
        const res = await testDatabaseConnection(db.id);
        return res;
      },
      {
        successMessage: (res) =>
          res.success
            ? `Connected to ${db.name} (${db.hostPort}) · ${res.latencyMs}ms`
            : `Connection to ${db.name} degraded`,
        successTone: 'pine',
        onError: () => {
          showSnackbar(`Connection to ${db.name} degraded`, 'maroon');
        },
      }
    );
    setTestingId(null);
  };

  const handleSaveDb = async (data: SaveDbInput) => {
    await runAsync(
      async () => {
        const saved = await saveDatabase({
          id: editingDb ? editingDb.id : undefined,
          name: data.name,
          engine: data.engine,
          hostPort: `${data.host}:${data.port}`,
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
