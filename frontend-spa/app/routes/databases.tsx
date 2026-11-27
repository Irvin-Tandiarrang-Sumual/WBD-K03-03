import React from 'react';
import { useLoaderData, Link } from 'react-router';
import { ArrowRight, Database, Edit3, Eye } from 'lucide-react';
import { EmptyState } from '../components/common/EmptyState';
import { PageHeader } from '../components/common/PageHeader';
import { SearchInput } from '../components/common/SearchInput';
import { QuarantineBanner } from '../components/feedback/QuarantineBanner';
import { useDatabasesCatalog } from '../hooks/useDatabasesCatalog';
import { databaseService } from '../services/database_service';
import { draftService } from '../services/draft_service';
import { groupService } from '../services/group_service';
import { migrationService } from '../services/migration_service';
import { formatMigrationVersion } from '../utils/formatters';
import { getPermissionBadgeClass } from '../utils/permissions';
import type { AccessGroup, DbEngine, DbPermission, TargetDatabase } from '../types';

export async function clientLoader() {
  const [rawDatabases, groups] = await Promise.all([
    databaseService.getAccessibleDatabases(),
    groupService.getGroups().catch(() => []),
  ]);

  const databases = await Promise.all(
    rawDatabases.map(async (db) => {
      const [head, drafts] = await Promise.all([
        db.currentMigrationId
          ? migrationService.getMigrationById(db.id, db.currentMigrationId).catch(() => null)
          : Promise.resolve(null),
        draftService.getDraftsByDatabaseId(db.id).catch(() => []),
      ]);
      const quarantinedDraft = drafts.find(
        (d) => d.status === 'partial' || d.status === 'partially_reverted'
      );
      return {
        ...db,
        headVersion: head?.version ?? 0,
        quarantined: Boolean(quarantinedDraft),
        quarantineDraftId: quarantinedDraft?.id,
        status: quarantinedDraft ? ('Degraded' as const) : db.status,
      };
    })
  );

  return { databases, groups };
}

export default function DatabasesPage() {
  const { databases, groups } = useLoaderData() as {
    databases: TargetDatabase[];
    groups: AccessGroup[];
  };

  const d = useDatabasesCatalog(databases, groups);

  return (
    <section className="space-y-5">
      <PageHeader
        title="Database Catalog"
        subtitle="Manage schema versions, inspect migrations, and propose schema drafts"
        badge={{ label: `${d.accessibleDbs.length} accessible`, variant: 'neutral' }}
      />

      {d.quarantinedDb && (
        <QuarantineBanner
          databaseName={d.quarantinedDb.name}
          quarantineDraftId={d.quarantinedDb.quarantineDraftId}
        />
      )}

      <CatalogFilterBar
        search={d.search}
        onSearchChange={d.setSearch}
        permissionFilter={d.permissionFilter}
        onPermissionFilterChange={d.setPermissionFilter}
        engineFilter={d.engineFilter}
        onEngineFilterChange={d.setEngineFilter}
      />

      {d.filteredDbs.length === 0 ? (
        <EmptyState
          title="No Databases Found"
          description="No databases match your search and filter criteria."
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {d.filteredDbs.map(({ database, groupName, permission }) => (
            <DatabaseCatalogCard
              key={database.id}
              database={database}
              groupName={groupName}
              permission={permission}
            />
          ))}
        </div>
      )}
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface AccessibleDatabaseItem {
  database: TargetDatabase;
  groupName: string;
  permission: DbPermission;
}

const DatabaseCatalogCard: React.FC<AccessibleDatabaseItem> = ({
  database,
  groupName,
  permission,
}) => {
  const permissionClass = getPermissionBadgeClass(permission);

  return (
    <article
      data-testid="catalog-card"
      className="h-[240px] flex flex-col justify-between p-6 bg-[#FFFDF9]/92 backdrop-blur-md border border-[#D8C7A5] rounded-2xl shadow-sm hover:shadow-md hover:-translate-y-1 transition duration-200"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2.5 min-w-0">
          <span className="w-8 h-8 rounded-lg bg-[#264D3B]/12 text-[#264D3B] flex items-center justify-center shrink-0">
            <Database className="w-4 h-4" />
          </span>
          <h2 className="font-mono text-sm font-bold text-[#3A241A] truncate">
            {database.name}
          </h2>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <span className="px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/25">
            {database.engine}
          </span>
          {database.quarantined && (
            <span className="px-2 py-0.5 rounded-full bg-[#822433] text-white font-mono text-[10px] font-bold">
              423
            </span>
          )}
        </div>
      </div>

      <div className="space-y-2.5">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-[#EFE5D1] text-[#3A241A] border border-[#D8C7A5]">
            {groupName}
          </span>
          <span
            className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${permissionClass}`}
          >
            {permission === 'view_edit' ? (
              <Edit3 className="w-3 h-3" />
            ) : (
              <Eye className="w-3 h-3" />
            )}
            <span>{permission}</span>
          </span>
        </div>

        <div className="text-xs text-[#6B5444] space-y-1">
          <div className="font-mono text-[11px] truncate">{database.hostPort}</div>
          <div className="font-mono text-[11px] text-[#264D3B]">
            schema version: {formatMigrationVersion(database.headVersion)}
          </div>
        </div>
      </div>

      <div className="pt-3 border-t border-[#EFE5D1] flex items-center justify-between">
        <span className="text-[11px] text-[#6B5444]">
          HEAD version #{database.headVersion}
        </span>
        <Link
          to={`/databases/${database.id}`}
          className="text-xs font-bold text-[#1E6B8C] hover:text-[#0F4A63] inline-flex items-center gap-1 transition cursor-pointer"
        >
          <span>Open Workspace</span>
          <ArrowRight className="w-3.5 h-3.5" />
        </Link>
      </div>
    </article>
  );
};

interface CatalogFilterBarProps {
  search: string;
  onSearchChange: (value: string) => void;
  permissionFilter: 'all' | DbPermission;
  onPermissionFilterChange: (value: 'all' | DbPermission) => void;
  engineFilter: 'all' | DbEngine;
  onEngineFilterChange: (value: 'all' | DbEngine) => void;
}

const CatalogFilterBar: React.FC<CatalogFilterBarProps> = ({
  search,
  onSearchChange,
  permissionFilter,
  onPermissionFilterChange,
  engineFilter,
  onEngineFilterChange,
}) => (
  <div className="flex flex-wrap items-center justify-between gap-3 p-3.5 bg-[#FFFDF9]/80 backdrop-blur-md rounded-2xl border border-[#D8C7A5]">
    <SearchInput
      value={search}
      onChange={onSearchChange}
      placeholder="Search databases..."
      className="max-w-sm"
    />

    <div className="flex items-center gap-2 flex-wrap">
      <select
        aria-label="Filter catalog by permission"
        value={permissionFilter}
        onChange={(e) => onPermissionFilterChange(e.target.value as 'all' | DbPermission)}
        className="min-h-[44px] h-11 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3.5 focus:outline-none focus:border-[#1E6B8C] focus-visible:ring-2 focus-visible:ring-[#264D3B] cursor-pointer"
      >
        <option value="all">Access: All</option>
        <option value="view_edit">view_edit</option>
        <option value="view_only">view_only</option>
      </select>
      <select
        aria-label="Filter catalog by engine"
        value={engineFilter}
        onChange={(e) => onEngineFilterChange(e.target.value as 'all' | DbEngine)}
        className="min-h-[44px] h-11 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3.5 focus:outline-none focus:border-[#1E6B8C] focus-visible:ring-2 focus-visible:ring-[#264D3B] cursor-pointer"
      >
        <option value="all">Engine: All</option>
        <option value="PostgreSQL">PostgreSQL</option>
        <option value="MySQL">MySQL</option>
      </select>
    </div>
  </div>
);
