import type { AccessGroup, DbPermission, TargetDatabase } from '../types';

export interface AccessibleDatabaseResult {
  database: TargetDatabase;
  groupName: string;
  permission: DbPermission;
}

export function getUserAccessibleDatabases(
  user: { id: string; role: 'admin' | 'user' },
  databases: TargetDatabase[],
  groups: AccessGroup[]
): AccessibleDatabaseResult[] {
  if (user.role === 'admin') {
    return mapAdminAccessibleDatabases(databases, groups);
  }
  return filterUserAccessibleDatabases(user.id, databases, groups);
}

export function getPermissionChangeConfirmation(
  dbName: string,
  nextPermission: DbPermission
): { title: string; question: string; warning: string } {
  const isGrantingEdit = nextPermission === 'view_edit';

  return {
    title: 'Update Database Access Permission?',
    question: `Are you sure you want to change access permission for ${dbName}?`,
    warning: isGrantingEdit
      ? "Promoting access to 'view_edit' grants all members in this group full authority to propose and execute migrations against this database."
      : "Demoting access to 'view_only' prevents members of this group from proposing or applying migrations against this database.",
  };
}

// --- Stepdown Rule Helpers ---

function mapAdminAccessibleDatabases(
  databases: TargetDatabase[],
  groups: AccessGroup[]
): AccessibleDatabaseResult[] {
  return databases.map((database) => {
    const linkedGroup = groups.find((group) =>
      group.databaseAccess.some((access) => access.databaseId === database.id)
    );
    return {
      database,
      groupName: linkedGroup?.name ?? 'Admin Global',
      permission: 'view_edit' as DbPermission,
    };
  });
}

function filterUserAccessibleDatabases(
  userId: string,
  databases: TargetDatabase[],
  groups: AccessGroup[]
): AccessibleDatabaseResult[] {
  const userGroups = groups.filter((g) => g.memberIds.includes(userId));
  const results: AccessibleDatabaseResult[] = [];

  for (const db of databases) {
    const accessMatch = findHighestDatabasePermission(db.id, userGroups);
    if (accessMatch) {
      results.push({
        database: db,
        groupName: accessMatch.groupName,
        permission: accessMatch.permission,
      });
    }
  }

  return results;
}

function findHighestDatabasePermission(
  databaseId: string,
  userGroups: AccessGroup[]
): { groupName: string; permission: DbPermission } | null {
  let highestMatch: { groupName: string; permission: DbPermission } | null = null;

  for (const group of userGroups) {
    const access = group.databaseAccess.find((a) => a.databaseId === databaseId);
    if (!access) continue;

    if (!highestMatch || access.permission === 'view_edit') {
      highestMatch = { groupName: group.name, permission: access.permission };
    }
  }

  return highestMatch;
}

export function getPermissionBadgeClass(permission: DbPermission): string {
  return permission === 'view_edit'
    ? 'bg-[#264D3B]/12 text-[#264D3B] border border-[#264D3B]/25'
    : 'bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/25';
}
