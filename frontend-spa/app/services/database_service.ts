import { api } from './api_client';
import { cleanEntityId } from '~/utils/formatters';
import type { TargetDatabase, DbEngine, DbStatus, DbPermission, DatabaseQueryResult } from '~/types/database';

export interface BackendDatabaseResponse {
  id: number;
  port: number;
  host: string;
  user: string;
  type: string;
  name: string;
  accessType?: string;
  currentMigrationId?: number | null;
}

export function mapBackendDatabaseToTargetDatabase(
  db: BackendDatabaseResponse,
  overrides?: Partial<TargetDatabase>
): TargetDatabase {
  const isPg = db.type ? db.type.toLowerCase().includes('postgr') : true;
  const rawAccess = (db.accessType || overrides?.permission || 'view_edit').toLowerCase();
  const permission: DbPermission = rawAccess.includes('edit') ? 'view_edit' : 'view_only';

  return {
    id: String(db.id),
    name: db.name,
    engine: (isPg ? 'PostgreSQL' : 'MySQL') as DbEngine,
    driverVersion: isPg ? 'PostgreSQL 16.4' : 'MySQL 8.0.39',
    host: db.host,
    port: db.port,
    hostPort: `${db.host}:${db.port}`,
    status: 'Healthy' as DbStatus,
    quarantined: false,
    headVersion: overrides?.headVersion ?? 1,
    currentMigrationId: db.currentMigrationId == null ? undefined : String(db.currentMigrationId),
    redoStack: overrides?.redoStack ?? [],
    permission,
    ...overrides,
  };
}

export const databaseService = {
  async getAccessibleDatabases(): Promise<TargetDatabase[]> {
    try {
      return await this.getUserDatabases();
    } catch {
      return await this.getAdminDatabases();
    }
  },

  async getAccessibleDatabaseById(id: string | number): Promise<TargetDatabase> {
    try {
      return await this.getUserDatabaseById(id);
    } catch {
      return await this.getAdminDatabaseById(id);
    }
  },

  async getAdminDatabases(params: { page?: number; limit?: number } = { page: 1, limit: 100 }): Promise<TargetDatabase[]> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendDatabaseResponse[] | { data: BackendDatabaseResponse[] }>(`/admin/databases${qs}`);
    const items: BackendDatabaseResponse[] = Array.isArray(res)
      ? res
      : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map((db) => mapBackendDatabaseToTargetDatabase(db));
  },

  async getAdminDatabaseById(id: string | number): Promise<TargetDatabase> {
    const rawId = cleanEntityId(id, 'db-');
    const res = await api.get<BackendDatabaseResponse>(`/admin/databases/${rawId}`);
    return mapBackendDatabaseToTargetDatabase(res);
  },

  async createAdminDatabase(payload: {
    name: string;
    type: 'postgresql' | 'mysql';
    host: string;
    port: number;
    user: string;
    password?: string;
  }): Promise<TargetDatabase> {
    const res = await api.post<BackendDatabaseResponse>('/admin/databases', {
      name: payload.name,
      type: payload.type.toLowerCase(),
      host: payload.host,
      port: Number(payload.port),
      user: payload.user || 'postgres',
      password: payload.password || '',
    });
    return mapBackendDatabaseToTargetDatabase(res);
  },

  async deleteAdminDatabase(id: string | number): Promise<void> {
    const rawId = cleanEntityId(id, 'db-');
    await api.delete(`/admin/databases/${rawId}`);
  },

  async getUserDatabases(params: { page?: number; limit?: number } = { page: 1, limit: 100 }): Promise<TargetDatabase[]> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendDatabaseResponse[] | { data: BackendDatabaseResponse[] }>(`/me/databases${qs}`);
    const items: BackendDatabaseResponse[] = Array.isArray(res)
      ? res
      : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map((db) => mapBackendDatabaseToTargetDatabase(db));
  },

  async getUserDatabaseById(id: string | number): Promise<TargetDatabase> {
    const rawId = cleanEntityId(id, 'db-');
    const res = await api.get<BackendDatabaseResponse>(`/me/databases/${rawId}`);
    return mapBackendDatabaseToTargetDatabase(res);
  },

  async testConnection(id: string | number): Promise<{ success: boolean; latencyMs: number }> {
    const rawId = cleanEntityId(id, 'db-');
    const start = performance.now();
    try {
      await api.get(`/admin/databases/${rawId}`);
      const latencyMs = Math.round(performance.now() - start);
      return { success: true, latencyMs };
    } catch {
      try {
        await api.get(`/me/databases/${rawId}`);
        const latencyMs = Math.round(performance.now() - start);
        return { success: true, latencyMs };
      } catch {
        const latencyMs = Math.round(performance.now() - start);
        return { success: false, latencyMs };
      }
    }
  },

  async executeQuery(id: string | number, querySql: string): Promise<DatabaseQueryResult> {
    const rawId = cleanEntityId(id, 'db-');
    const start = performance.now();
    const res = await api.post<DatabaseQueryResult>(`/me/databases/${rawId}/query`, { query: querySql });
    const latencyMs = Math.round(performance.now() - start);
    return { ...res, latencyMs };
  },

  async explainQuery(id: string | number, querySql: string): Promise<DatabaseQueryResult> {
    const rawId = cleanEntityId(id, 'db-');
    const start = performance.now();
    const res = await api.post<DatabaseQueryResult>(`/me/databases/${rawId}/explain`, { query: querySql });
    const latencyMs = Math.round(performance.now() - start);
    return { ...res, latencyMs };
  },

  async explainAnalyzeQuery(id: string | number, querySql: string): Promise<DatabaseQueryResult> {
    const rawId = cleanEntityId(id, 'db-');
    const start = performance.now();
    const res = await api.post<DatabaseQueryResult>(`/me/databases/${rawId}/explain-analyze`, { query: querySql });
    const latencyMs = Math.round(performance.now() - start);
    return { ...res, latencyMs };
  },
};
