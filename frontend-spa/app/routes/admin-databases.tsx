import React from 'react';
import { useLoaderData } from 'react-router';
import { Edit3, Plus, Wifi } from 'lucide-react';
import { ColumnsDropdown } from '../components/common/ColumnsDropdown';
import { PageHeader } from '../components/common/PageHeader';
import { SearchInput } from '../components/common/SearchInput';
import { RegisterDbModal } from '../components/modals/RegisterDbModal';
import { useAdminDatabases } from '../hooks/useAdminDatabases';
import { fetchDatabases } from '../services/dummy_databases';
import { fetchGroups } from '../services/dummy_groups';
import type { AccessGroup, DbEngine, DbStatus, TargetDatabase } from '../types';

export async function clientLoader() {
  const [databases, groups] = await Promise.all([fetchDatabases(), fetchGroups()]);
  return { databases, groups };
}

export default function AdminDatabasesPage() {
  const { databases: initialDatabases, groups } = useLoaderData() as {
    databases: TargetDatabase[];
    groups: AccessGroup[];
  };

  const d = useAdminDatabases(initialDatabases);

  return (
    <section className="space-y-5">
      <PageHeader
        title="Databases Registry"
        subtitle="Manage database connections, TLS parameters, and health probes"
        actions={
          <button
            type="button"
            onClick={() => {
              d.setEditingDb(null);
              d.setRegisterModalOpen(true);
            }}
            className="h-10 px-4 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            <Plus className="w-4 h-4" />
            <span>Register Database</span>
          </button>
        }
      />

      <AdminDatabasesTable
        databases={d.filteredDatabases}
        groups={groups}
        search={d.search}
        onSearchChange={d.setSearch}
        engineFilter={d.engineFilter}
        onEngineFilterChange={d.setEngineFilter}
        statusFilter={d.statusFilter}
        onStatusFilterChange={d.setStatusFilter}
        columnsMap={d.columnsMap}
        onToggleColumn={d.toggleColumn}
        testingId={d.testingId}
        onTestConnection={d.handleTestConnection}
        onEdit={(item) => {
          d.setEditingDb(item);
          d.setRegisterModalOpen(true);
        }}
      />

      <RegisterDbModal
        open={d.registerModalOpen}
        editingDb={d.editingDb}
        onClose={() => {
          d.setRegisterModalOpen(false);
          d.setEditingDb(null);
        }}
        onSave={d.handleSaveDb}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface AdminDatabasesTableProps {
  databases: TargetDatabase[];
  groups: AccessGroup[];
  search: string;
  onSearchChange: (value: string) => void;
  engineFilter: 'all' | DbEngine;
  onEngineFilterChange: (value: 'all' | DbEngine) => void;
  statusFilter: 'all' | DbStatus;
  onStatusFilterChange: (value: 'all' | DbStatus) => void;
  columnsMap: Record<string, boolean>;
  onToggleColumn: (key: string) => void;
  testingId: string | null;
  onTestConnection: (db: TargetDatabase) => void;
  onEdit: (db: TargetDatabase) => void;
}

const AdminDatabasesTable: React.FC<AdminDatabasesTableProps> = ({
  databases,
  groups,
  search,
  onSearchChange,
  engineFilter,
  onEngineFilterChange,
  statusFilter,
  onStatusFilterChange,
  columnsMap,
  onToggleColumn,
  testingId,
  onTestConnection,
  onEdit,
}) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
    <AdminDatabasesFilterBar
      search={search}
      onSearchChange={onSearchChange}
      engineFilter={engineFilter}
      onEngineFilterChange={onEngineFilterChange}
      statusFilter={statusFilter}
      onStatusFilterChange={onStatusFilterChange}
      columnsMap={columnsMap}
      onToggleColumn={onToggleColumn}
    />

    <div className="overflow-x-auto">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-[#EFE5D1] text-[11px] font-bold uppercase tracking-wider text-[#6B5444] bg-[#F5EEDD]/40">
            <th className="py-3 px-4">DB Name</th>
            {columnsMap.engine && <th className="py-3 px-4">Engine</th>}
            {columnsMap.hostPort && <th className="py-3 px-4">Host:Port</th>}
            {columnsMap.groups && <th className="py-3 px-4">Linked Groups</th>}
            {columnsMap.status && <th className="py-3 px-4">Health</th>}
            <th className="py-3 px-4 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#EFE5D1] text-xs">
          {databases.map((db) => {
            const linkedCount = groups.filter((g) =>
              g.databaseAccess.some((acc) => acc.databaseId === db.id)
            ).length;
            return (
              <AdminDatabaseTableRow
                key={db.id}
                database={db}
                linkedCount={linkedCount}
                columnsMap={columnsMap}
                isTesting={testingId === db.id}
                onTestConnection={onTestConnection}
                onEdit={onEdit}
              />
            );
          })}
        </tbody>
      </table>
    </div>
  </div>
);

interface AdminDatabasesFilterBarProps {
  search: string;
  onSearchChange: (value: string) => void;
  engineFilter: 'all' | DbEngine;
  onEngineFilterChange: (value: 'all' | DbEngine) => void;
  statusFilter: 'all' | DbStatus;
  onStatusFilterChange: (value: 'all' | DbStatus) => void;
  columnsMap: Record<string, boolean>;
  onToggleColumn: (key: string) => void;
}

const AdminDatabasesFilterBar: React.FC<AdminDatabasesFilterBarProps> = ({
  search,
  onSearchChange,
  engineFilter,
  onEngineFilterChange,
  statusFilter,
  onStatusFilterChange,
  columnsMap,
  onToggleColumn,
}) => (
  <div className="p-4 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3">
    <SearchInput
      value={search}
      onChange={onSearchChange}
      placeholder="Search databases..."
      className="max-w-xs"
    />
    <div className="flex flex-wrap items-center gap-2">
      <select
        aria-label="Filter databases by engine"
        value={engineFilter}
        onChange={(e) => onEngineFilterChange(e.target.value as 'all' | DbEngine)}
        className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
      >
        <option value="all">Engine: All</option>
        <option value="PostgreSQL">PostgreSQL</option>
        <option value="MySQL">MySQL</option>
      </select>
      <select
        aria-label="Filter databases by status"
        value={statusFilter}
        onChange={(e) => onStatusFilterChange(e.target.value as 'all' | DbStatus)}
        className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
      >
        <option value="all">Status: All</option>
        <option value="Healthy">Healthy</option>
        <option value="Degraded">Degraded</option>
      </select>
      <ColumnsDropdown
        columns={[
          { key: 'engine', label: 'Engine', visible: columnsMap.engine },
          { key: 'hostPort', label: 'Host:Port', visible: columnsMap.hostPort },
          { key: 'groups', label: 'Linked Groups', visible: columnsMap.groups },
          { key: 'status', label: 'Health Status', visible: columnsMap.status },
        ]}
        onToggle={onToggleColumn}
      />
    </div>
  </div>
);

interface AdminDatabaseTableRowProps {
  database: TargetDatabase;
  linkedCount: number;
  columnsMap: Record<string, boolean>;
  isTesting: boolean;
  onTestConnection: (db: TargetDatabase) => void;
  onEdit: (db: TargetDatabase) => void;
}

const AdminDatabaseTableRow: React.FC<AdminDatabaseTableRowProps> = ({
  database,
  linkedCount,
  columnsMap,
  isTesting,
  onTestConnection,
  onEdit,
}) => {
  const isHealthy = database.status === 'Healthy';
  const statusColorClass = isHealthy ? 'text-[#264D3B]' : 'text-[#8A5A00]';
  const statusDotClass = isHealthy ? 'bg-[#264D3B]' : 'bg-[#F3B738]';

  return (
    <tr className="hover:bg-[#F5EEDD]/45 transition-colors">
      <td className="py-3.5 px-4">
        <div className="flex items-center gap-2">
          <span className="font-mono font-semibold text-[#3A241A]">{database.name}</span>
          {database.quarantined && (
            <span className="px-2 py-0.5 rounded-full bg-[#822433] text-white text-[10px] font-bold">
              423
            </span>
          )}
        </div>
      </td>
      {columnsMap.engine && (
        <td className="py-3.5 px-4">
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/25">
            {database.engine}
          </span>
        </td>
      )}
      {columnsMap.hostPort && (
        <td className="py-3.5 px-4 font-mono text-xs text-[#6B5444]">
          {database.hostPort}
        </td>
      )}
      {columnsMap.groups && (
        <td className="py-3.5 px-4 text-xs font-semibold text-[#3A241A]">
          {linkedCount} {linkedCount === 1 ? 'group' : 'groups'}
        </td>
      )}
      {columnsMap.status && (
        <td className="py-3.5 px-4">
          <span className="inline-flex items-center gap-1.5 text-xs font-semibold">
            <span className={`w-2.5 h-2.5 rounded-full ${statusDotClass}`} />
            <span className={statusColorClass}>{database.status}</span>
          </span>
        </td>
      )}
      <td className="py-3.5 px-4 text-right">
        <div className="inline-flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => onTestConnection(database)}
            disabled={isTesting}
            className="h-8 px-3 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <Wifi className={`w-3.5 h-3.5 text-[#1E6B8C] ${isTesting ? 'animate-pulse' : ''}`} />
            <span>{isTesting ? 'Testing...' : 'Test Connection'}</span>
          </button>
          <button
            type="button"
            onClick={() => onEdit(database)}
            className="h-8 px-2.5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1 transition-colors cursor-pointer"
          >
            <Edit3 className="w-3.5 h-3.5 text-[#264D3B]" />
            <span>Edit</span>
          </button>
        </div>
      </td>
    </tr>
  );
};
