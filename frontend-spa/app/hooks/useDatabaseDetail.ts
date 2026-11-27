import { useMemo, useRef, useState } from 'react';
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

  const database = initialDatabase;
  const migrations = initialMigrations;

  const [filters, setFilters] = useState<FilterState>({
    search: '',
    hideRollbacks: false,
    status: 'all',
  });

  const [columnsMap, setColumnsMap] = useState<Record<string, boolean>>(DEFAULT_COLUMNS);

  const [selectedMigration, setSelectedMigration] = useState<MigrationRecord | null>(null);
  const [diffMigration, setDiff] = useState<MigrationRecord | null>(null);
  const diffRequest = useRef(0);
  const [confirmRollbackOpen, setConfirmRollbackOpen] = useState(false);

  // Database permission is directly granted by backend access control
  const permission: DbPermission =
    database.permission ?? (user.role === 'admin' ? 'view_edit' : 'view_only');

  const rollbackCheck = canRollbackDatabase(database, migrations, permission);
  const redoCheck = canForwardRedoDatabase(database, migrations, permission);
  const headMig = getHeadMigration(database, migrations);

  const filteredMigrations = useMemo(
    () => filterMigrations(migrations, filters.search, filters.hideRollbacks, filters.status),
    [migrations, filters.search, filters.hideRollbacks, filters.status]
  );

  const handleRollback = async () => {
    setConfirmRollbackOpen(false);
    await runAsync(
      () => migrationService.proposeRollback(database.id),
      {
        successMessage: () => `Rollback draft created / proposed successfully`,
        successTone: 'pine',
        errorMessage: 'Rollback failed',
      }
    );
  };

  const handleRedo = async () => {
    await runAsync(
      () => migrationService.proposeForward(database.id),
      {
        successMessage: () => `Forward draft created / proposed successfully`,
        successTone: 'pine',
        errorMessage: 'Forward failed',
      }
    );
  };

  const toggleColumn = (key: string) => {
    setColumnsMap((prev) => toggleColumnVisibility(prev, key, 1));
  };

  const setDiffMigration = async (migration: MigrationRecord | null) => {
    const request = ++diffRequest.current;
    if (!migration) {
      setDiff(null);
      return;
    }
    const index = migrations.findIndex((entry) => entry.id === migration.id);
    const previous = migrations.slice(0, index).reverse().find((entry) => entry.status === 'Success');
    if (!previous || migration.status !== 'Success') {
      setDiff({ ...migration, schemaDiff: [{ type: 'context', content: 'No comparable schema snapshot.' }] });
      return;
    }
    await runAsync(
      () => migrationService.compareMigrationSnapshots(database.id, previous, migration),
      {
        errorMessage: 'Failed to compare schema snapshots',
        onSuccess: (updated) => {
          if (request === diffRequest.current) setDiff(updated);
        },
      }
    );
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
