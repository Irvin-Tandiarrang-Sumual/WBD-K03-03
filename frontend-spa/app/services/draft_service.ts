import { api, getApiBaseUrl } from './api_client';
import { databaseService } from './database_service';
import { cleanEntityId } from '~/utils/formatters';
import type { TargetDatabase } from '~/types/database';
import type {
  DraftProposal,
  DraftStatus,
  LeaderConsensusVote,
  LeaderVoteStatus,
  BatchConfig,
} from '~/types/draft';

export interface BackendDraftApproval {
  id: number;
  draftId: number;
  groupId: number;
  groupName?: string;
  leaderId?: number | null;
  leaderUsername?: string;
  status: string;
  round: number;
  phase?: string;
  vote?: string;
  respondedAt?: string | null;
}

export interface BackendDraftResponse {
  id: number;
  databaseId: number;
  userId: number;
  username?: string;
  operation?: string;
  targetId?: number;
  baseExecutionId?: number;
  sqlUp: string;
  sqlDown: string;
  isReversible: boolean;
  irreversibleReason?: string;
  isBatch: boolean;
  batchColumn?: string;
  batchSize?: number;
  currentRound: number;
  recoveryAction?: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  approvals: BackendDraftApproval[];
}

export interface BackendBatchProgress {
  current_batch: number;
  total_batches: number;
  rows_processed: number;
  total_rows: number;
  percentage: number;
  status: string;
}

export function mapBackendDraftToProposal(
  d: BackendDraftResponse,
  progress?: BackendBatchProgress
): DraftProposal {
  const votes: LeaderConsensusVote[] = (d.approvals || []).map((a) => {
    const rawVote = (a.vote || a.status || 'pending').toLowerCase();
    let voteStatus: LeaderVoteStatus = 'pending';
    if (['approved', 'rejected', 'resume', 'revert', 'yes', 'no'].includes(rawVote)) {
      voteStatus = rawVote as LeaderVoteStatus;
    }

    const leaderDisplayName = a.leaderUsername || (a.leaderId ? `Leader #${a.leaderId}` : `Group #${a.groupId} Leader`);
    const groupDisplayName = a.groupName || `Group #${a.groupId}`;

    return {
      leaderId: a.leaderId ? String(a.leaderId) : `leader-grp-${a.groupId}`,
      leaderName: leaderDisplayName,
      groupName: groupDisplayName,
      status: voteStatus,
      votedAt: a.respondedAt || undefined,
      round: a.round,
      phase: a.phase,
    };
  });

  let batchConfig: BatchConfig | undefined;
  if (d.isBatch) {
    const currentBatch = progress?.current_batch ?? 0;
    const totalBatches = progress?.total_batches ?? 5;
    const processedRows = progress?.rows_processed ?? 0;
    const totalRows = progress?.total_rows ?? 2500;

    const matchTable = d.sqlUp.match(/(?:FROM|UPDATE|INTO)\s+([a-zA-Z0-9_]+)/i);
    const tableName = matchTable ? matchTable[1] : 'batch_table';

    batchConfig = {
      targetTable: tableName,
      batchColumn: d.batchColumn || 'id',
      batchSize: d.batchSize || 500,
      processedRows: Number(processedRows),
      totalRows: Number(totalRows),
      currentBatch,
      totalBatches,
    };
  }

  let validStatus: DraftStatus = 'pending_approval';
  const rawStatus = (d.status || '').toLowerCase();
  if (rawStatus === 'approved') validStatus = 'approved';
  else if (rawStatus === 'applied') validStatus = 'applied';
  else if (rawStatus === 'executing') validStatus = 'executing';
  else if (rawStatus === 'partial') validStatus = 'partial';
  else if (rawStatus === 'partially_reverted') validStatus = 'partially_reverted';
  else if (rawStatus === 'rejected') validStatus = 'rejected';
  else if (rawStatus === 'failed') validStatus = 'failed';
  else if (rawStatus === 'invalidated') validStatus = 'invalidated';

  const desc = d.sqlUp.split('\n')[0].replace(/^--\s*/, '').slice(0, 50) || `Proposal #${d.id}`;

  return {
    id: String(d.id),
    databaseId: String(d.databaseId),
    description: desc,
    proposerId: String(d.userId),
    proposerName: d.username || (d.userId ? `User #${d.userId}` : 'System Developer'),
    mode: d.isBatch ? 'batch_dml' : 'normal',
    status: validStatus,
    operation: d.operation,
    targetId: d.targetId,
    baseExecutionId: d.baseExecutionId,
    upSql: d.sqlUp || '',
    downSql: d.sqlDown || undefined,
    batchConfig,
    consensus: votes,
    updatedRelative: 'just now',
    failureMessage: d.recoveryAction ? `Recovery action: ${d.recoveryAction}` : undefined,
    currentRound: d.currentRound,
    recoveryAction: d.recoveryAction,
  };
}

export const draftService = {
  async getDraftsByDatabaseId(
    databaseId: string | number,
    params: { page?: number; limit?: number } = { page: 1, limit: 100 }
  ): Promise<DraftProposal[]> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendDraftResponse[] | { data: BackendDraftResponse[] }>(
      `/databases/${rawDbId}/drafts${qs}`
    );
    const items: BackendDraftResponse[] = Array.isArray(res)
      ? res
      : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map((d) => mapBackendDraftToProposal(d));
  },

  async getAllDraftsAcrossDatabases(databases?: TargetDatabase[]): Promise<DraftProposal[]> {
    const dbs = databases || (await databaseService.getAccessibleDatabases());
    const draftPromises = dbs.map((db) =>
      this.getDraftsByDatabaseId(db.id).catch(() => [])
    );
    const draftsNested = await Promise.all(draftPromises);
    return draftsNested.flat();
  },

  async getAllDrafts(): Promise<DraftProposal[]> {
    return this.getAllDraftsAcrossDatabases();
  },

  async getDraftById(id: string | number): Promise<DraftProposal> {
    const rawId = cleanEntityId(id, 'DRF-');
    let progress: BackendBatchProgress | undefined;
    try {
      progress = await api.get<BackendBatchProgress>(`/drafts/${rawId}/progress`);
    } catch {
      // not batch or not executing
    }
    const res = await api.get<BackendDraftResponse>(`/drafts/${rawId}`);
    return mapBackendDraftToProposal(res, progress);
  },

  async createDraft(
    databaseId: string | number,
    payload: {
      sqlUp: string;
      sqlDown?: string;
      isReversible?: boolean;
      irreversibleReason?: string;
      isBatch?: boolean;
      batchColumn?: string;
      batchSize?: number;
    }
  ): Promise<DraftProposal> {
    const rawDbId = cleanEntityId(databaseId, 'db-');
    const res = await api.post<BackendDraftResponse>(`/databases/${rawDbId}/drafts`, {
      sqlUp: payload.sqlUp,
      sqlDown: payload.sqlDown || '',
      isReversible: payload.isReversible ?? true,
      irreversibleReason: payload.irreversibleReason || '',
      isBatch: !!payload.isBatch,
      batchColumn: payload.batchColumn || '',
      batchSize: payload.batchSize || 500,
    });
    return mapBackendDraftToProposal(res);
  },

  async submitApproval(draftId: string | number, status: string): Promise<DraftProposal> {
    const rawId = cleanEntityId(draftId, 'DRF-');
    const res = await api.post<BackendDraftResponse>(`/drafts/${rawId}/approvals`, {
      status,
    });
    return mapBackendDraftToProposal(res);
  },

  async executeDraft(draftId: string | number): Promise<void> {
    const rawId = cleanEntityId(draftId, 'DRF-');
    await api.post(`/drafts/${rawId}/execute`);
  },

  async resumeBatch(draftId: string | number): Promise<void> {
    const rawId = cleanEntityId(draftId, 'DRF-');
    await api.post(`/drafts/${rawId}/resume`);
  },

  async revertBatch(draftId: string | number): Promise<void> {
    const rawId = cleanEntityId(draftId, 'DRF-');
    await api.post(`/drafts/${rawId}/revert-batch`);
  },

  async getBatchProgress(draftId: string | number): Promise<BackendBatchProgress> {
    const rawId = cleanEntityId(draftId, 'DRF-');
    return await api.get<BackendBatchProgress>(`/drafts/${rawId}/progress`);
  },

  streamBatchProgress(
    draftId: string | number,
    onUpdate: (progress: BackendBatchProgress) => void,
    onComplete?: () => void,
    onError?: (err: any) => void
  ): () => void {
    if (typeof window === 'undefined' || typeof EventSource === 'undefined') {
      return () => {};
    }

    const rawId = cleanEntityId(draftId, 'DRF-');
    const url = `${getApiBaseUrl()}/drafts/${rawId}/progress/stream`;
    const eventSource = new EventSource(url, { withCredentials: true });

    eventSource.onmessage = (event) => {
      try {
        const data: BackendBatchProgress = JSON.parse(event.data);
        onUpdate(data);

        const isTerminal = ['applied', 'failed', 'partial', 'partially_reverted'].includes(
          (data.status || '').toLowerCase()
        );
        if (isTerminal) {
          eventSource.close();
          if (onComplete) onComplete();
        }
      } catch (err) {
        console.error('[SSE] Failed to parse batch progress event', err);
      }
    };

    eventSource.onerror = (err) => {
      eventSource.close();
      if (onError) onError(err);
    };

    return () => {
      eventSource.close();
    };
  },
};
