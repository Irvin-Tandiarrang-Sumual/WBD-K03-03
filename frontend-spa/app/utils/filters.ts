import type { MigrationRecord, MigrationStatusFilter } from '../types';
import { formatMigrationVersion } from './formatters';

export function filterMigrations(
  migrations: MigrationRecord[],
  query = '',
  hideRollbacks = false,
  status: MigrationStatusFilter = 'all'
): MigrationRecord[] {
  const normalizedQuery = query.trim().toLowerCase();
  const hiddenIds = new Set<string>();
  if (hideRollbacks) {
    for (const migration of migrations) {
      if (migration.status === 'Success' && migration.revertsMigrationId) {
        hiddenIds.add(migration.id);
        hiddenIds.add(migration.revertsMigrationId);
      }
    }
  }

  return migrations.filter((migration) => {
    if (hiddenIds.has(migration.id)) return false;
    if (status !== 'all' && migration.status !== status) return false;
    if (normalizedQuery && !matchesMigrationQuery(migration, normalizedQuery)) return false;
    return true;
  });
}

function matchesMigrationQuery(migration: MigrationRecord, query: string): boolean {
  return (
    formatMigrationVersion(migration.version).toLowerCase().includes(query) ||
    migration.summary.toLowerCase().includes(query) ||
    migration.executedBy.toLowerCase().includes(query) ||
    migration.upSql.toLowerCase().includes(query)
  );
}
