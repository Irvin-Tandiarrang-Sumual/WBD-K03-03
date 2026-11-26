import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { migrationService } from '../services/migration_service';
import {
  canForwardRedoDatabase,
  canRollbackDatabase,
  getHeadMigration,
} from '../utils/databaseRules';
import { filterMigrations } from '../utils/filters';
import { toggleColumnVisibility } from '../utils/table';
import type {
  AccessGroup,
  DbPermission,
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
  hideRollbacks: boolean;
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
    hideRollbacks: false,
    status: 'all',
  });

  const [columnsMap, setColumnsMap] = useState<Record<string, boolean>>(DEFAULT_COLUMNS);

  const [selectedMigration, setSelectedMigration] = useState<MigrationRecord | null>(null);
  const [diffMigration, setDiffMigration] = useState<MigrationRecord | null>(null);
  const [confirmRollbackOpen, setConfirmRollbackOpen] = useState(false);

  // Database permission is directly granted by backend access control
  const permission: DbPermission =
    database.permission ?? (user.role === 'admin' ? 'view_edit' : 'view_only');

  const rollbackCheck = canRollbackDatabase(database, migrations, permission);
  const redoCheck = canForwardRedoDatabase(database, permission);
  const headMig = getHeadMigration(database, migrations);

  const filteredMigrations = useMemo(
    () => filterMigrations(migrations, filters.search, filters.hideRollbacks, filters.status),
    [migrations, filters.search, filters.hideRollbacks, filters.status]
  );

  const handleRollback = async () => {
    setConfirmRollbackOpen(false);
    await runAsync(
      async () => {
        const res = await migrationService.proposeRollback(database.id);
        const updatedMigrations = await migrationService.getMigrationsByDatabaseId(database.id);
        return { res, updatedMigrations };
      },
      {
        successMessage: () => `Rollback draft created / proposed successfully`,
        successTone: 'pine',
        errorMessage: 'Rollback failed',
        onSuccess: async ({ updatedMigrations }) => {
          setMigrations(updatedMigrations);
          if (updatedMigrations.length > 0) {
            const nextHead = Math.max(...updatedMigrations.map((m) => m.version));
            setDatabase((prev) => ({ ...prev, headVersion: nextHead }));
          }
        },
      }
    );
  };

  const handleRedo = async () => {
    await runAsync(
      async () => {
        const res = await migrationService.proposeForward(database.id);
        const updatedMigrations = await migrationService.getMigrationsByDatabaseId(database.id);
        return { res, updatedMigrations };
      },
      {
        successMessage: () => `Forward draft created / proposed successfully`,
        successTone: 'pine',
        errorMessage: 'Forward failed',
        onSuccess: async ({ updatedMigrations }) => {
          setMigrations(updatedMigrations);
          if (updatedMigrations.length > 0) {
            const nextHead = Math.max(...updatedMigrations.map((m) => m.version));
            setDatabase((prev) => ({ ...prev, headVersion: nextHead }));
          }
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
    hideRollbacks: filters.hideRollbacks,
    setHideRollbacks: (hideRollbacks: boolean) => setFilters((prev) => ({ ...prev, hideRollbacks })),
    statusFilter: filters.status,
    setStatusFilter: (status: MigrationStatusFilter) => setFilters((prev) => ({ ...prev, status })),
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
