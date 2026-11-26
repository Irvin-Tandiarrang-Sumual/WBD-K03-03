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
  createdAt?: string;
}

export function mapBackendMigrationToRecord(m: BackendMigrationResponse): MigrationRecord {
  const versionNum = parseInt(m.version, 10) || m.id || 1;
  const isRollback = m.revertsMigrationId !== undefined && m.revertsMigrationId !== null;
  const isFailed = (m.status || '').toLowerCase().includes('fail');

  const diffLines: SchemaDiffLine[] = [
    { type: 'header', content: `--- Schema State before v${versionNum}` },
    { type: 'header', content: `+++ Schema State after v${versionNum}` },
    { type: 'context', content: `@@ migration executed: ${m.version} @@` },
  ];

  if (m.sqlUp) {
    const lines = m.sqlUp.split('\n');
    lines.forEach((l) => {
      const trimmed = l.trim();
      if (trimmed.startsWith('--') || trimmed === '') {
        diffLines.push({ type: 'context', content: `  ${l}` });
      } else {
        diffLines.push({ type: 'add', content: `+ ${l}` });
      }
    });
  }

  return {
    id: String(m.id),
    databaseId: String(m.databaseId),
    version: versionNum,
    summary: m.sqlUp.slice(0, 60).replace(/\n/g, ' ') || `Migration v${versionNum}`,
    type: isRollback ? 'ROLLBACK' : 'UP',
    status: isFailed ? 'Fail' : 'Success',
    reversible: m.isReversible ?? true,
    executedBy: m.userId ? `User #${m.userId}` : 'System Admin',
    executedAt: m.createdAt || new Date().toISOString(),
    executedAtRelative: 'recently',
    durationMs: 45,
    upSql: m.sqlUp || '',
    downSql: m.sqlDown || undefined,
    errorLog: m.errorMessage || undefined,
    diffFromVersion: Math.max(0, versionNum - 1),
    schemaDiff: diffLines,
  };
}

export const migrationService = {
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
  ): Promise<any> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.get(
      `/me/databases/${rawDbId}/migrations/compare?migration_id_1=${migrationId1}&migration_id_2=${migrationId2}`
    );
  },

  async proposeRollback(databaseId: string | number): Promise<any> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.post(`/me/databases/${rawDbId}/rollback-draft`);
  },

  async proposeForward(databaseId: string | number): Promise<any> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    return await api.post(`/me/databases/${rawDbId}/forward-draft`);
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
