import { useMemo, useState } from 'react';
import { toggleColumnVisibility } from '../utils/table';
import type { DraftProposal, DraftStatus } from '../types';

export function useDraftList(initialDrafts: DraftProposal[]) {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | DraftStatus>('all');
  const [databaseFilter, setDatabaseFilter] = useState<string>('all');

  const [columnsMap, setColumnsMap] = useState<Record<string, boolean>>({
    database: true,
    mode: true,
    proposer: true,
    consensus: true,
    status: true,
    updated: true,
  });

  const filteredDrafts = useMemo(() => {
    return initialDrafts.filter((d) => {
      if (
        search &&
        !d.id.toLowerCase().includes(search.toLowerCase()) &&
        !d.description.toLowerCase().includes(search.toLowerCase())
      ) {
        return false;
      }
      if (statusFilter !== 'all' && d.status !== statusFilter) {
        return false;
      }
      if (databaseFilter !== 'all' && d.databaseId !== databaseFilter) {
        return false;
      }
      return true;
    });
  }, [initialDrafts, search, statusFilter, databaseFilter]);

  const toggleColumn = (key: string) => {
    setColumnsMap((prev) => toggleColumnVisibility(prev, key, 1));
  };

  return {
    search,
    setSearch,
    statusFilter,
    setStatusFilter,
    databaseFilter,
    setDatabaseFilter,
    columnsMap,
    toggleColumn,
    filteredDrafts,
  };
}
