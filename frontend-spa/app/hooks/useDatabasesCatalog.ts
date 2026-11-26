import { useMemo, useState } from 'react';
import type { AccessGroup, DbEngine, DbPermission, TargetDatabase } from '../types';

export function useDatabasesCatalog(databases: TargetDatabase[], groups: AccessGroup[] = []) {
  const [search, setSearch] = useState('');
  const [permissionFilter, setPermissionFilter] = useState<'all' | DbPermission>('all');
  const [engineFilter, setEngineFilter] = useState<'all' | DbEngine>('all');

  // Databases and permissions are already authorized and provided by backend
  const accessibleDbs = useMemo(() => {
    return databases.map((database) => {
      const permission: DbPermission = database.permission || 'view_edit';
      const linkedGroup = groups.find((g) =>
        g.databaseAccess?.some((a) => a.databaseId === database.id)
      );
      return {
        database,
        groupName: database.groupName || linkedGroup?.name || 'Assigned Group',
        permission,
      };
    });
  }, [databases, groups]);

  const filteredDbs = useMemo(() => {
    return accessibleDbs.filter(({ database, permission }) => {
      if (search && !database.name.toLowerCase().includes(search.toLowerCase())) {
        return false;
      }
      if (permissionFilter !== 'all' && permission !== permissionFilter) {
        return false;
      }
      if (engineFilter !== 'all' && database.engine !== engineFilter) {
        return false;
      }
      return true;
    });
  }, [accessibleDbs, search, permissionFilter, engineFilter]);

  const quarantinedDb = databases.find((d) => d.quarantined);

  return {
    search,
    setSearch,
    permissionFilter,
    setPermissionFilter,
    engineFilter,
    setEngineFilter,
    accessibleDbs,
    filteredDbs,
    quarantinedDb,
  };
}
