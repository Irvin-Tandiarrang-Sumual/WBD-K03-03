import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { getUserAccessibleDatabases } from '../utils/permissions';
import type { AccessGroup, DbEngine, DbPermission, TargetDatabase } from '../types';

export function useDatabasesCatalog(databases: TargetDatabase[], groups: AccessGroup[]) {
  const { user } = useAuth();

  const [search, setSearch] = useState('');
  const [permissionFilter, setPermissionFilter] = useState<'all' | DbPermission>('all');
  const [engineFilter, setEngineFilter] = useState<'all' | DbEngine>('all');

  const accessibleDbs = useMemo(
    () => getUserAccessibleDatabases(user, databases, groups),
    [user, databases, groups]
  );

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
