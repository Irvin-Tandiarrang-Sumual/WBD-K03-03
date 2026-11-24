import React from 'react';
import { useLoaderData, useNavigate } from 'react-router';
import { ColumnsDropdown } from '../components/common/ColumnsDropdown';
import { DraftStatusChip } from '../components/common/DraftStatusChip';
import { PageHeader } from '../components/common/PageHeader';
import { SearchInput } from '../components/common/SearchInput';
import { useDraftList } from '../hooks/useDraftList';
import { fetchDatabases } from '../services/dummy_databases';
import { fetchDrafts } from '../services/dummy_drafts';
import type { DraftProposal, DraftStatus, TargetDatabase } from '../types';

export async function clientLoader() {
  const [drafts, databases] = await Promise.all([fetchDrafts(), fetchDatabases()]);
  return { drafts, databases };
}

export default function DraftListPage() {
  const { drafts, databases } = useLoaderData() as {
    drafts: DraftProposal[];
    databases: TargetDatabase[];
  };
  const navigate = useNavigate();
  const d = useDraftList(drafts);

  return (
    <section className="space-y-5">
      <PageHeader
        title="Schema Drafts"
        subtitle="Consensus proposals, migrations in progress, and historical executions"
        badge={{ label: `${drafts.length} total`, variant: 'neutral' }}
      />

      <DraftListTable
        drafts={d.filteredDrafts}
        databases={databases}
        search={d.search}
        onSearchChange={d.setSearch}
        databaseFilter={d.databaseFilter}
        onDatabaseFilterChange={d.setDatabaseFilter}
        statusFilter={d.statusFilter}
        onStatusFilterChange={d.setStatusFilter}
        columnsMap={d.columnsMap}
        onToggleColumn={d.toggleColumn}
        onSelectDraft={(id) => navigate(`/drafts/${id}`)}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface DraftListTableProps {
  drafts: DraftProposal[];
  databases: TargetDatabase[];
  search: string;
  onSearchChange: (value: string) => void;
  databaseFilter: string;
  onDatabaseFilterChange: (value: string) => void;
  statusFilter: 'all' | DraftStatus;
  onStatusFilterChange: (value: 'all' | DraftStatus) => void;
  columnsMap: Record<string, boolean>;
  onToggleColumn: (key: string) => void;
  onSelectDraft: (id: string) => void;
}

const DraftListTable: React.FC<DraftListTableProps> = ({
  drafts,
  databases,
  search,
  onSearchChange,
  databaseFilter,
  onDatabaseFilterChange,
  statusFilter,
  onStatusFilterChange,
  columnsMap,
  onToggleColumn,
  onSelectDraft,
}) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
    <DraftFilterToolbar
      search={search}
      onSearchChange={onSearchChange}
      databaseFilter={databaseFilter}
      onDatabaseFilterChange={onDatabaseFilterChange}
      statusFilter={statusFilter}
      onStatusFilterChange={onStatusFilterChange}
      databases={databases}
      columnsMap={columnsMap}
      onToggleColumn={onToggleColumn}
    />

    <div className="overflow-x-auto">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-[#EFE5D1] text-[11px] font-bold uppercase tracking-wider text-[#6B5444] bg-[#F5EEDD]/40">
            <th className="py-3 px-4">Draft</th>
            {columnsMap.database && <th className="py-3 px-4">Database</th>}
            {columnsMap.mode && <th className="py-3 px-4">Mode</th>}
            {columnsMap.proposer && <th className="py-3 px-4">Author</th>}
            {columnsMap.consensus && <th className="py-3 px-4">Consensus</th>}
            {columnsMap.status && <th className="py-3 px-4">Status</th>}
            {columnsMap.updated && <th className="py-3 px-4 text-right">Updated</th>}
          </tr>
        </thead>
        <tbody className="divide-y divide-[#EFE5D1] text-sm">
          {drafts.length === 0 ? (
            <tr>
              <td
                colSpan={7}
                className="py-12 text-center text-xs font-semibold text-[#8A7361]"
              >
                No drafts found matching the current criteria.
              </td>
            </tr>
          ) : (
            drafts.map((draft) => {
              const db = databases.find((d) => d.id === draft.databaseId);
              return (
                <DraftTableRow
                  key={draft.id}
                  draft={draft}
                  database={db}
                  columnsMap={columnsMap}
                  onSelect={onSelectDraft}
                />
              );
            })
          )}
        </tbody>
      </table>
    </div>
  </div>
);

interface DraftFilterToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  databaseFilter: string;
  onDatabaseFilterChange: (value: string) => void;
  statusFilter: 'all' | DraftStatus;
  onStatusFilterChange: (value: 'all' | DraftStatus) => void;
  databases: TargetDatabase[];
  columnsMap: Record<string, boolean>;
  onToggleColumn: (key: string) => void;
}

const DraftFilterToolbar: React.FC<DraftFilterToolbarProps> = ({
  search,
  onSearchChange,
  databaseFilter,
  onDatabaseFilterChange,
  statusFilter,
  onStatusFilterChange,
  databases,
  columnsMap,
  onToggleColumn,
}) => (
  <div className="p-4 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3">
    <SearchInput
      value={search}
      onChange={onSearchChange}
      placeholder="Search drafts..."
      className="max-w-xs"
    />

    <div className="flex flex-wrap items-center gap-2">
      <select
        aria-label="Filter drafts by target database"
        value={databaseFilter}
        onChange={(e) => onDatabaseFilterChange(e.target.value)}
        className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
      >
        <option value="all">Target DB: All</option>
        {databases.map((db) => (
          <option key={db.id} value={db.id}>
            {db.name}
          </option>
        ))}
      </select>

      <select
        aria-label="Filter drafts by consensus status"
        value={statusFilter}
        onChange={(e) => onStatusFilterChange(e.target.value as 'all' | DraftStatus)}
        className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
      >
        <option value="all">Status: All</option>
        <option value="pending_approval">Pending Approval</option>
        <option value="approved">Approved</option>
        <option value="executing">Executing</option>
        <option value="applied">Applied</option>
        <option value="partially_reverted">Partially Reverted</option>
        <option value="rejected">Rejected</option>
        <option value="failed">Failed</option>
      </select>

      <ColumnsDropdown
        columns={[
          { key: 'database', label: 'Database', visible: columnsMap.database },
          { key: 'mode', label: 'Execution Mode', visible: columnsMap.mode },
          { key: 'proposer', label: 'Author', visible: columnsMap.proposer },
          { key: 'consensus', label: 'Quorum Ratio', visible: columnsMap.consensus },
          { key: 'status', label: 'Consensus Status', visible: columnsMap.status },
          { key: 'updated', label: 'Last Updated', visible: columnsMap.updated },
        ]}
        onToggle={onToggleColumn}
      />
    </div>
  </div>
);

interface DraftTableRowProps {
  draft: DraftProposal;
  database?: TargetDatabase;
  columnsMap: Record<string, boolean>;
  onSelect: (id: string) => void;
}

const DraftTableRow: React.FC<DraftTableRowProps> = ({
  draft,
  database,
  columnsMap,
  onSelect,
}) => {
  const approvedCount = draft.consensus.filter((c) => c.status === 'approved').length;
  const modeLabel = draft.mode === 'batch_dml' ? 'Batch DML' : 'Normal';

  return (
    <tr
      onClick={() => onSelect(draft.id)}
      className="hover:bg-[#F5EEDD]/55 transition-colors cursor-pointer"
    >
      <td className="py-3.5 px-4">
        <div className="font-mono text-xs font-bold text-[#1E6B8C]">{draft.id}</div>
        <div className="text-xs font-medium text-[#3A241A] mt-0.5 truncate max-w-[240px]">
          {draft.description}
        </div>
      </td>
      {columnsMap.database && (
        <td className="py-3.5 px-4 font-mono text-xs text-[#3A241A]">
          {database?.name ?? draft.databaseId}
        </td>
      )}
      {columnsMap.mode && (
        <td className="py-3.5 px-4">
          <span className="px-2 py-0.5 rounded-md text-[11px] font-semibold bg-[#EFE5D1] text-[#3A241A]">
            {modeLabel}
          </span>
        </td>
      )}
      {columnsMap.proposer && (
        <td className="py-3.5 px-4 font-mono text-xs text-[#6B5444]">
          {draft.proposerName}
        </td>
      )}
      {columnsMap.consensus && (
        <td className="py-3.5 px-4 font-mono text-xs font-bold text-[#3A241A]">
          {approvedCount}/{draft.consensus.length}
        </td>
      )}
      {columnsMap.status && (
        <td className="py-3.5 px-4">
          <DraftStatusChip status={draft.status} />
        </td>
      )}
      {columnsMap.updated && (
        <td className="py-3.5 px-4 text-right text-xs font-mono text-[#6B5444]">
          {draft.updatedRelative}
        </td>
      )}
    </tr>
  );
};
