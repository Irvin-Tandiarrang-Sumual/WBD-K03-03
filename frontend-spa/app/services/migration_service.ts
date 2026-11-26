import { api } from './api_client';
import { cleanEntityId } from '~/utils/formatters';
import type { MigrationRecord, SchemaDiffLine } from '~/types/migration';

export interface BackendMigrationResponse {
  id: number;
  databaseId: number;
  version: string;
  status: string;
  sqlUp: string;
  sqlDown: string;
  errorMessage?: string;
  executionId?: number;
  revertsMigrationId?: number | null;
  sourceMigrationId?: number | null;
  isReversible?: boolean;
  irreversibleReason?: string;
  userId?: number;
  migratedAt?: string;
  createdAt?: string;
}

export interface BackendMigrationDraftResponse {
  id: number;
  operation: string;
  targetMigrationId: number;
  sqlUp: string;
  status: string;
}

export function mapBackendMigrationToRecord(m: BackendMigrationResponse): MigrationRecord {
  const versionNum = parseInt(m.version, 10) || m.id || 1;
  const isRollback = m.revertsMigrationId !== undefined && m.revertsMigrationId !== null;
  const isFailed = (m.status || '').toLowerCase().includes('fail');
  const executedAt = m.migratedAt ?? m.createdAt;

  return {
    id: String(m.id),
    revertsMigrationId: m.revertsMigrationId == null ? undefined : String(m.revertsMigrationId),
    sourceMigrationId: m.sourceMigrationId == null ? undefined : String(m.sourceMigrationId),
    databaseId: String(m.databaseId),
    version: versionNum,
    summary: m.sqlUp.slice(0, 60).replace(/\n/g, ' ') || `Migration v${versionNum}`,
    type: isRollback ? 'ROLLBACK' : 'UP',
    status: isFailed ? 'Fail' : 'Success',
    reversible: m.isReversible ?? true,
    executedBy: m.userId ? `User #${m.userId}` : 'System Admin',
    executedAt,
    executedAtRelative: executedAt ? new Date(executedAt).toLocaleString() : '—',
    durationMs: 45,
    upSql: m.sqlUp || '',
    downSql: m.sqlDown || undefined,
    errorLog: m.errorMessage || undefined,
    diffFromVersion: Math.max(0, versionNum - 1),
    schemaDiff: [],
  };
}

export const migrationService = {
  async getMigrationHistory(databaseId: string | number): Promise<MigrationRecord[]> {
    const limit = 100;
    const history: MigrationRecord[] = [];
    for (let page = 1; ; page++) {
      const migrations = await this.getMigrationsByDatabaseId(databaseId, { page, limit });
      history.push(...migrations);
      if (migrations.length < limit) return history;
    }
  },

  async getMigrationsByDatabaseId(
    databaseId: string | number,
    params: { page?: number; limit?: number; hideRollback?: boolean } = { page: 1, limit: 100 }
  ): Promise<MigrationRecord[]> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    if (params.hideRollback) query.set('hide_rollback', 'true');
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendMigrationResponse[] | { data: BackendMigrationResponse[] }>(
      `/me/databases/${rawDbId}/migrations${qs}`
    );
    const items = Array.isArray(res) ? res : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map(mapBackendMigrationToRecord);
  },

  async getMigrationById(databaseId: string | number, migrationId: string | number): Promise<MigrationRecord> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const res = await api.get<BackendMigrationResponse>(
      `/me/databases/${rawDbId}/migrations/${migrationId}`
    );
    return mapBackendMigrationToRecord(res);
  },

  async compareMigrations(
    databaseId: string | number,
    migrationId1: string | number,
    migrationId2: string | number
  ): Promise<{ schemaDiff?: { isIdentical: boolean; summaryItems: { detail: string; type: string }[] } }> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.get(
      `/me/databases/${rawDbId}/migrations/compare?migration_id_1=${migrationId1}&migration_id_2=${migrationId2}`
    );
  },

  async compareMigrationSnapshots(databaseId: string, previous: MigrationRecord, current: MigrationRecord): Promise<MigrationRecord> {
    const { schemaDiff } = await this.compareMigrations(databaseId, previous.id, current.id);
    const lines: SchemaDiffLine[] = schemaDiff?.summaryItems.map((item) => ({
      type: item.type.includes('added') ? 'add' : item.type.includes('removed') ? 'remove' : 'context',
      content: item.detail,
    })) ?? [];
    return {
      ...current,
      diffFromVersion: previous.version,
      schemaDiff: lines.length ? lines : [{
        type: 'context',
        content: schemaDiff?.isIdentical ? 'No schema changes.' : 'No comparable schema snapshot.',
      }],
    };
  },

  async proposeRollback(databaseId: string | number): Promise<BackendMigrationDraftResponse> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.post<BackendMigrationDraftResponse>(`/me/databases/${rawDbId}/rollback-draft`);
  },

  async proposeForward(databaseId: string | number): Promise<BackendMigrationDraftResponse> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.post<BackendMigrationDraftResponse>(`/me/databases/${rawDbId}/forward-draft`);
  },

  async createMigration(databaseId: string | number, sqlUp: string): Promise<MigrationRecord> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const res = await api.post<BackendMigrationResponse>(`/me/databases/${rawDbId}/migrations`, {
      sqlUp,
    });
    return mapBackendMigrationToRecord(res);
  },

  async voteMigrationDraft(
    databaseId: string | number,
    draftId: string | number,
    approve: boolean
  ): Promise<void> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    await api.post(`/me/databases/${rawDbId}/migration-drafts/${draftId}/vote`, {
      approve,
    });
  },

  async executeMigrationDraft(
    databaseId: string | number,
    draftId: string | number
  ): Promise<MigrationRecord> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const res = await api.post<BackendMigrationResponse>(
      `/me/databases/${rawDbId}/migration-drafts/${draftId}/execute`
    );
    return mapBackendMigrationToRecord(res);
  },
};
