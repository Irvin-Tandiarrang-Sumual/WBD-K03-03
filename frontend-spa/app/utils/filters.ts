import type { MigrationRecord, MigrationStatusFilter } from '../types';
import { formatMigrationVersion } from './formatters';

export function filterMigrations(
  migrations: MigrationRecord[],
  query = '',
  hideRollbacks = false,
  status: MigrationStatusFilter = 'all'
): MigrationRecord[] {
  const normalizedQuery = query.trim().toLowerCase();

  return migrations.filter((migration) => {
    if (hideRollbacks && migration.type === 'ROLLBACK') return false;
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
