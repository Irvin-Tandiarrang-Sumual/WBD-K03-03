import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import {
  executeForwardRedo,
  executeRollback,
  fetchMigrationsByDatabaseId,
} from '../services/dummy_migrations';
import {
  canForwardRedoDatabase,
  canRollbackDatabase,
  getHeadMigration,
} from '../utils/databaseRules';
import { filterMigrations } from '../utils/filters';
import { getUserAccessibleDatabases } from '../utils/permissions';
import { toggleColumnVisibility } from '../utils/table';
import type {
  AccessGroup,
  MigrationDirectionFilter,
  MigrationRecord,
  MigrationStatusFilter,
  TargetDatabase,
} from '../types';
import { useAsyncAction } from './useAsyncAction';

export interface UseDatabaseDetailProps {
  initialDatabase: TargetDatabase;
  initialMigrations: MigrationRecord[];
  allDbs: TargetDatabase[];
  groups: AccessGroup[];
}

interface FilterState {
  search: string;
  direction: MigrationDirectionFilter;
  status: MigrationStatusFilter;
}

const DEFAULT_COLUMNS: Record<string, boolean> = {
  version: true,
  type: true,
  status: true,
  reversibility: true,
  executedAt: true,
  executedBy: true,
};

export function useDatabaseDetail({
  initialDatabase,
  initialMigrations,
  allDbs,
  groups,
}: UseDatabaseDetailProps) {
  const { user, showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [database, setDatabase] = useState<TargetDatabase>(initialDatabase);
  const [migrations, setMigrations] = useState<MigrationRecord[]>(initialMigrations);

  const [filters, setFilters] = useState<FilterState>({
    search: '',
    direction: 'all',
    status: 'all',
  });

  const [columnsMap, setColumnsMap] = useState<Record<string, boolean>>(DEFAULT_COLUMNS);

  const [selectedMigration, setSelectedMigration] = useState<MigrationRecord | null>(null);
  const [diffMigration, setDiffMigration] = useState<MigrationRecord | null>(null);
  const [confirmRollbackOpen, setConfirmRollbackOpen] = useState(false);

  const accessibleDbs = useMemo(
    () => getUserAccessibleDatabases(user, allDbs, groups),
    [user, allDbs, groups]
  );
  const currentAccess = accessibleDbs.find((a) => a.database.id === database.id);
  const permission = currentAccess?.permission ?? 'view_edit';

  const rollbackCheck = canRollbackDatabase(database, migrations, permission);
  const redoCheck = canForwardRedoDatabase(database, permission);
  const headMig = getHeadMigration(database, migrations);

  const filteredMigrations = useMemo(
    () => filterMigrations(migrations, filters.search, filters.direction, filters.status),
    [migrations, filters.search, filters.direction, filters.status]
  );

  const handleRollback = async () => {
    setConfirmRollbackOpen(false);
    await runAsync(
      async () => {
        const res = await executeRollback(database.id, user.username);
        if (!res.success) throw new Error('Rollback execution failed');
        return res;
      },
      {
        successMessage: (res) => `Rolled back to #${res.newHeadVersion}`,
        successTone: 'pine',
        errorMessage: 'Rollback failed',
        onSuccess: async (res) => {
          setDatabase((prev) => applyRollbackDatabaseState(prev, res.newHeadVersion));
          const updatedMigrations = await fetchMigrationsByDatabaseId(database.id);
          setMigrations(updatedMigrations);
        },
      }
    );
  };

  const handleRedo = async () => {
    if (!redoCheck.targetVersion) return;
    const targetVersion = redoCheck.targetVersion;

    await runAsync(
      async () => {
        const res = await executeForwardRedo(database.id, targetVersion, user.username);
        if (!res.success) throw new Error('Redo execution failed');
        return res;
      },
      {
        successMessage: (res) => `Redo applied forward to #${res.newHeadVersion}`,
        successTone: 'pine',
        errorMessage: 'Redo failed',
        onSuccess: async (res) => {
          setDatabase((prev) => applyRedoDatabaseState(prev, res.newHeadVersion));
          const updatedMigrations = await fetchMigrationsByDatabaseId(database.id);
          setMigrations(updatedMigrations);
        },
      }
    );
  };

  const toggleColumn = (key: string) => {
    setColumnsMap((prev) => toggleColumnVisibility(prev, key, 1));
  };

  return {
    database,
    migrations,
    filteredMigrations,
    headMig,
    permission,
    rollbackCheck,
    redoCheck,
    search: filters.search,
    setSearch: (search: string) => setFilters((prev) => ({ ...prev, search })),
    directionFilter: filters.direction,
    setDirectionFilter: (direction: MigrationDirectionFilter) =>
      setFilters((prev) => ({ ...prev, direction })),
    statusFilter: filters.status,
    setStatusFilter: (status: MigrationStatusFilter) =>
      setFilters((prev) => ({ ...prev, status })),
    columnsMap,
    toggleColumn,
    selectedMigration,
    setSelectedMigration,
    diffMigration,
    setDiffMigration,
    confirmRollbackOpen,
    setConfirmRollbackOpen,
    handleRollback,
    handleRedo,
  };
}

// --- Pure State Transition Helpers (Stepdown Rule) ---

function applyRollbackDatabaseState(prev: TargetDatabase, newHeadVersion: number): TargetDatabase {
  return {
    ...prev,
    headVersion: newHeadVersion,
    redoStack: [...prev.redoStack, prev.headVersion],
  };
}

function applyRedoDatabaseState(prev: TargetDatabase, newHeadVersion: number): TargetDatabase {
  return {
    ...prev,
    headVersion: newHeadVersion,
    redoStack: prev.redoStack.slice(0, -1),
  };
}
