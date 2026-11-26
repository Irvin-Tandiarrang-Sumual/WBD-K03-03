import type { DbPermission, MigrationRecord, TargetDatabase } from '../types';

export function getHeadMigration(
  database: TargetDatabase,
  migrations: MigrationRecord[]
): MigrationRecord | undefined {
  const headId = database.currentMigrationId;
  if (!headId) return undefined;
  const revertedIds = new Set(migrations.filter((m) => m.status === 'Success').map((m) => m.revertsMigrationId));
  return [...migrations].reverse().find((m) =>
    m.databaseId === database.id && m.status === 'Success' && m.type === 'UP' &&
    !revertedIds.has(m.id) && (m.id === headId || m.sourceMigrationId === headId)
  );
}

export function canRollbackDatabase(
  database: TargetDatabase,
  migrations: MigrationRecord[],
  userPermission: DbPermission
): { allowed: boolean; reason?: string } {
  if (userPermission !== 'view_edit') {
    return { allowed: false, reason: 'You have view-only access on this database' };
  }
  if (database.quarantined) {
    return { allowed: false, reason: 'Database is quarantined due to unresolved batch/revert failure' };
  }
  if (database.headVersion <= 0) {
    return { allowed: false, reason: 'Database is already at baseline version (#0)' };
  }

  const headMigration = getHeadMigration(database, migrations);
  if (!headMigration) {
    return { allowed: false, reason: 'No applied head migration found' };
  }
  if (!headMigration.reversible) {
    return { allowed: false, reason: `Migration #${headMigration.version} is irreversible: ${headMigration.summary}` };
  }

  return { allowed: true };
}

export function canForwardRedoDatabase(
  database: TargetDatabase,
  userPermission: DbPermission
): { allowed: boolean; targetVersion?: number; reason?: string } {
  if (userPermission !== 'view_edit') {
    return { allowed: false, reason: 'You have view-only access on this database' };
  }
  if (database.quarantined) {
    return { allowed: false, reason: 'Database is quarantined' };
  }
  return { allowed: true };
}
