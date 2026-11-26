import React from 'react';
import { useLoaderData, useNavigate, Link } from 'react-router';
import {
  CheckCircle2,
  FileCode2,
  FileDiff,
  Plus,
  RotateCcw,
  RotateCw,
  Terminal,
  XCircle,
} from 'lucide-react';
import { ColumnsDropdown } from '../components/common/ColumnsDropdown';
import { PageHeader } from '../components/common/PageHeader';
import { SearchInput } from '../components/common/SearchInput';
import { ConfirmDialog } from '../components/feedback/ConfirmDialog';
import { QuarantineBanner } from '../components/feedback/QuarantineBanner';
import { MigrationDetailModal } from '../components/modals/MigrationDetailModal';
import { SchemaDiffModal } from '../components/modals/SchemaDiffModal';
import { useDatabaseDetail } from '../hooks/useDatabaseDetail';
import { databaseService } from '../services/database_service';
import { groupService } from '../services/group_service';
import { migrationService } from '../services/migration_service';
import { formatMigrationVersion } from '../utils/formatters';
import type {
  AccessGroup,
  MigrationRecord,
  MigrationStatusFilter,
  TargetDatabase,
} from '../types';

export async function clientLoader({ params }: { params: { id?: string } }) {
  const dbId = params.id || '1';
  let database: TargetDatabase | null = null;
  try {
    database = await databaseService.getAccessibleDatabaseById(dbId);
  } catch {
    throw new Response('Target Database Not Found', { status: 404 });
  }

  const [migrations, allDbs, groups] = await Promise.all([
    migrationService.getMigrationHistory(dbId),
    databaseService.getAccessibleDatabases(),
    groupService.getGroups().catch(() => []),
  ]);

  if (!database) {
    throw new Response('Target Database Not Found', { status: 404 });
  }

  const headId = database.currentMigrationId;
  const head = migrations.find((migration) => migration.id === headId);
  database = { ...database, headVersion: head?.version ?? 0 };

  return { database, migrations, allDbs, groups };
}

export default function DatabaseDetailPage() {
  const { database, migrations, allDbs, groups } = useLoaderData() as {
    database: TargetDatabase;
    migrations: MigrationRecord[];
    allDbs: TargetDatabase[];
    groups: AccessGroup[];
  };

  const navigate = useNavigate();
  const d = useDatabaseDetail({ initialDatabase: database, initialMigrations: migrations, allDbs, groups });

  return (
    <section className="space-y-5">
      <PageHeader
        title={`${d.database.name} Workspace`}
        subtitle={`${d.database.engine} · ${d.database.hostPort} · Role: ${d.permission}`}
        badge={{
          label: d.database.quarantined ? '423 QUARANTINED' : d.database.status.toUpperCase(),
          variant: d.database.quarantined ? 'danger' : 'neutral',
        }}
        actions={
          <div className="flex items-center gap-2">
            <Link
              to={`/console?db=${d.database.id}`}
              className="h-10 px-3.5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
            >
              <Terminal className="w-4 h-4 text-[#1E6B8C]" />
              <span>SQL Console</span>
            </Link>
            <select
              aria-label="Switch active target database"
              value={d.database.id}
              onChange={(e) => navigate(`/databases/${e.target.value}`)}
              className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
            >
              {allDbs.map((db) => (
                <option key={db.id} value={db.id}>
                  {db.name} ({db.engine})
                </option>
              ))}
            </select>
          </div>
        }
      />

      {d.database.quarantined && (
        <QuarantineBanner databaseName={d.database.name} quarantineDraftId={d.database.quarantineDraftId} />
      )}

      <DatabaseTraversalControls
        database={d.database}
        rollbackCheck={d.rollbackCheck}
        redoCheck={d.redoCheck}
        permission={d.permission}
        onRequestRollback={() => d.setConfirmRollbackOpen(true)}
        onRequestRedo={d.handleRedo}
      />

      <MigrationTable
        migrations={d.filteredMigrations}
        headMigrationId={d.headMig?.id}
        search={d.search}
        onSearchChange={d.setSearch}
        hideRollbacks={d.hideRollbacks}
        onHideRollbacksChange={d.setHideRollbacks}
        statusFilter={d.statusFilter}
        onStatusFilterChange={d.setStatusFilter}
        columnsMap={d.columnsMap}
        onToggleColumn={d.toggleColumn}
        onSelectMigration={d.setSelectedMigration}
        onOpenDiff={d.setDiffMigration}
      />

      <MigrationDetailModal
        migration={d.selectedMigration}
        onClose={() => d.setSelectedMigration(null)}
        onOpenDiff={(m) => {
          d.setSelectedMigration(null);
          d.setDiffMigration(m);
        }}
      />

      <SchemaDiffModal
        migration={d.diffMigration}
        onClose={() => d.setDiffMigration(null)}
      />

      <ConfirmDialog
        open={d.confirmRollbackOpen}
        onClose={() => d.setConfirmRollbackOpen(false)}
        onConfirm={d.handleRollback}
        title="Execute 1-Step Linear Rollback?"
        message={`Are you sure you want to rollback HEAD on ${d.database.name} from #${d.database.headVersion} to #${
          d.database.headVersion - 1
        }? This will execute the recorded down migration script.`}
        confirmText="Confirm Rollback"
        isDestructive={true}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface DatabaseTraversalControlsProps {
  database: TargetDatabase;
  rollbackCheck: { allowed: boolean; reason?: string };
  redoCheck: { allowed: boolean; reason?: string };
  permission: string;
  onRequestRollback: () => void;
  onRequestRedo: () => void;
}

const DatabaseTraversalControls: React.FC<DatabaseTraversalControlsProps> = ({
  database,
  rollbackCheck,
  redoCheck,
  permission,
  onRequestRollback,
  onRequestRedo,
}) => {
  const isCreateDisabled = permission !== 'view_edit' || database.quarantined;

  return (
    <div className="p-4 bg-[#FFFDF9]/90 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs flex flex-wrap items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-2.5">
        <button
          type="button"
          disabled={!rollbackCheck.allowed}
          onClick={onRequestRollback}
          title={rollbackCheck.reason}
          className="min-h-[44px] h-11 px-4 rounded-xl bg-[#822433] hover:bg-[#681C28] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer disabled:opacity-45 disabled:cursor-not-allowed focus-visible:ring-2 focus-visible:ring-[#822433]"
        >
          <RotateCcw className="w-4 h-4" />
          <span>Rollback (1-Step)</span>
        </button>

        <button
          type="button"
          disabled={!redoCheck.allowed}
          onClick={onRequestRedo}
          title={redoCheck.reason}
          className="min-h-[44px] h-11 px-4 rounded-xl bg-[#1E6B8C] hover:bg-[#16536E] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer disabled:opacity-45 disabled:cursor-not-allowed focus-visible:ring-2 focus-visible:ring-[#1E6B8C]"
        >
          <RotateCw className="w-4 h-4" />
          <span>Forward (Redo)</span>
        </button>

        <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#264D3B]/12 border border-[#264D3B]/30 text-[#264D3B] font-mono text-xs font-bold">
          <span className="w-2 h-2 rounded-full bg-[#264D3B]" />
          HEAD · {formatMigrationVersion(database.headVersion)}
        </span>

        {!rollbackCheck.allowed && rollbackCheck.reason && (
          <span className="text-xs font-medium text-[#6B5444] hidden sm:inline">
            ({rollbackCheck.reason})
          </span>
        )}
      </div>

      <Link
        to={isCreateDisabled ? '#' : `/databases/${database.id}/drafts/new`}
        className={`min-h-[44px] h-11 px-4 rounded-xl text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B] ${
          isCreateDisabled
            ? 'bg-[#A39281] text-white cursor-not-allowed opacity-60'
            : 'bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] cursor-pointer'
        }`}
      >
        <Plus className="w-4 h-4" />
        <span>Propose Migration Draft</span>
      </Link>
    </div>
  );
};

interface MigrationTableProps {
  migrations: MigrationRecord[];
  headMigrationId?: string;
  search: string;
  onSearchChange: (value: string) => void;
  hideRollbacks: boolean;
  onHideRollbacksChange: (hide: boolean) => void;
  statusFilter: MigrationStatusFilter;
  onStatusFilterChange: (value: MigrationStatusFilter) => void;
  columnsMap: Record<string, boolean>;
  onToggleColumn: (key: string) => void;
  onSelectMigration: (migration: MigrationRecord) => void;
  onOpenDiff: (migration: MigrationRecord) => void;
}

const MigrationTable: React.FC<MigrationTableProps> = ({
  migrations,
  headMigrationId,
  search,
  onSearchChange,
  hideRollbacks,
  onHideRollbacksChange,
  statusFilter,
  onStatusFilterChange,
  columnsMap,
  onToggleColumn,
  onSelectMigration,
  onOpenDiff,
}) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs">
    <div className="p-4 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3 rounded-t-2xl relative z-20">
      <SearchInput
        value={search}
        onChange={onSearchChange}
        placeholder="Search history..."
        className="max-w-xs"
      />
      <div className="flex flex-wrap items-center gap-2">
        <label className="min-h-[44px] h-11 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-2 cursor-pointer transition-colors select-none focus-within:ring-2 focus-within:ring-[#264D3B]">
          <input
            type="checkbox"
            checked={hideRollbacks}
            onChange={(e) => onHideRollbacksChange(e.target.checked)}
            className="w-4 h-4 rounded border-[#D8C7A5] text-[#264D3B] focus:ring-[#264D3B] accent-[#264D3B] cursor-pointer"
          />
          <span>Hide Rollbacks</span>
        </label>
        <select
          aria-label="Filter migrations by status"
          value={statusFilter}
          onChange={(e) => onStatusFilterChange(e.target.value as MigrationStatusFilter)}
          className="min-h-[44px] h-11 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3.5 focus:outline-none focus:border-[#1E6B8C] focus-visible:ring-2 focus-visible:ring-[#264D3B] cursor-pointer"
        >
          <option value="all">Status: All</option>
          <option value="Success">Success</option>
          <option value="Fail">Fail</option>
        </select>
        <ColumnsDropdown
          columns={[
            { key: 'version', label: 'Version', visible: columnsMap.version },
            { key: 'type', label: 'Direction Type', visible: columnsMap.type },
            { key: 'status', label: 'Execution Status', visible: columnsMap.status },
            { key: 'reversibility', label: 'Reversibility', visible: columnsMap.reversibility },
            { key: 'executedAt', label: 'Executed Timestamp', visible: columnsMap.executedAt },
            { key: 'executedBy', label: 'Author / Operator', visible: columnsMap.executedBy },
          ]}
          onToggle={onToggleColumn}
        />
      </div>
    </div>

    <div className="overflow-x-auto rounded-b-2xl">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-[#EFE5D1] text-[11px] font-bold uppercase tracking-wider text-[#6B5444] bg-[#F5EEDD]/40">
            {columnsMap.version && <th className="py-3 px-4">Version</th>}
            {columnsMap.type && <th className="py-3 px-4">Type</th>}
            {columnsMap.status && <th className="py-3 px-4">Status</th>}
            {columnsMap.reversibility && <th className="py-3 px-4">Reversibility</th>}
            {columnsMap.executedAt && <th className="py-3 px-4">Executed At</th>}
            {columnsMap.executedBy && <th className="py-3 px-4">Executed By</th>}
            <th className="py-3 px-4 text-right">Inspection</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#EFE5D1] text-xs">
          {migrations.map((m) => (
            <MigrationTableRow
              key={m.id}
              migration={m}
              isHead={m.id === headMigrationId}
              columnsMap={columnsMap}
              onSelectMigration={onSelectMigration}
              onOpenDiff={onOpenDiff}
            />
          ))}
        </tbody>
      </table>
    </div>
  </div>
);

interface MigrationTableRowProps {
  migration: MigrationRecord;
  isHead: boolean;
  columnsMap: Record<string, boolean>;
  onSelectMigration: (m: MigrationRecord) => void;
  onOpenDiff: (m: MigrationRecord) => void;
}

const MigrationTableRow: React.FC<MigrationTableRowProps> = ({
  migration,
  isHead,
  columnsMap,
  onSelectMigration,
  onOpenDiff,
}) => {
  const isFailed = migration.status === 'Fail';
  const rowClass = getMigrationRowClass(isFailed, isHead);

  const typeClass =
    migration.type === 'UP'
      ? 'bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/30'
      : 'bg-[#822433]/12 text-[#822433] border border-[#822433]/30';

  const reversibilityClass = migration.reversible
    ? 'bg-[#264D3B]/12 text-[#264D3B] border border-[#264D3B]/25'
    : 'bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/65';

  return (
    <tr className={`transition-colors ${rowClass}`}>
      {columnsMap.version && (
        <td className="py-3 px-4 whitespace-nowrap">
          <div className="inline-flex items-center gap-2">
            <span className="font-mono text-xs font-bold text-[#3A241A]">
              {formatMigrationVersion(migration.version)}
            </span>
            {isHead && (
              <span className="px-2 py-0.5 rounded-full bg-[#264D3B] text-[#F5EEDD] text-[10px] font-bold">
                HEAD
              </span>
            )}
          </div>
        </td>
      )}
      {columnsMap.type && (
        <td className="py-3 px-4 whitespace-nowrap">
          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold ${typeClass}`}>
            {migration.type}
          </span>
        </td>
      )}
      {columnsMap.status && (
        <td className="py-3 px-4 whitespace-nowrap">
          <span
            className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-bold ${
              isFailed
                ? 'bg-[#822433]/15 text-[#822433] border border-[#822433]/30'
                : 'bg-[#264D3B]/12 text-[#264D3B] border border-[#264D3B]/25'
            }`}
          >
            {isFailed ? <XCircle className="w-3 h-3" /> : <CheckCircle2 className="w-3 h-3" />}
            <span>{migration.status}</span>
          </span>
        </td>
      )}
      {columnsMap.reversibility && (
        <td className="py-3 px-4 whitespace-nowrap">
          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${reversibilityClass}`}>
            {migration.reversible ? 'Reversible' : 'Irreversible'}
          </span>
        </td>
      )}
      {columnsMap.executedAt && (
        <td className="py-3 px-4 whitespace-nowrap font-mono text-xs text-[#6B5444]">
          {migration.executedAtRelative}
        </td>
      )}
      {columnsMap.executedBy && (
        <td className="py-3 px-4 whitespace-nowrap font-mono text-xs text-[#3A241A]">
          {migration.executedBy}
        </td>
      )}
      <td className="py-3 px-4 text-right whitespace-nowrap">
        <div className="inline-flex items-center justify-end gap-1.5">
          <button
            type="button"
            onClick={() => onSelectMigration(migration)}
            className="h-8 px-2.5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1 transition-colors cursor-pointer"
          >
            <FileCode2 className="w-3.5 h-3.5 text-[#1E6B8C]" />
            <span>SQL</span>
          </button>
          <button
            type="button"
            onClick={() => onOpenDiff(migration)}
            className="h-8 px-2.5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1 transition-colors cursor-pointer"
          >
            <FileDiff className="w-3.5 h-3.5 text-[#264D3B]" />
            <span>Diff</span>
          </button>
        </div>
      </td>
    </tr>
  );
};

function getMigrationRowClass(isFailed: boolean, isHead: boolean): string {
  if (isFailed) return 'bg-[#822433]/[0.05] hover:bg-[#822433]/[0.09]';
  if (isHead) return 'bg-[#264D3B]/[0.05] hover:bg-[#264D3B]/[0.09]';
  return 'hover:bg-[#F5EEDD]/45';
}
