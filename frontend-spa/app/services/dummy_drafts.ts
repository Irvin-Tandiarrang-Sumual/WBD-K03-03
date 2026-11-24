import type { DraftProposal, LeaderVoteStatus, LeaderConsensusVote, BatchConfig } from '../types';
import type { MigrationRecord } from '../types';
import { appendMigration } from './dummy_migrations';
import { updateDatabaseHeadVersion, updateDatabaseQuarantine } from './dummy_databases';

export const INITIAL_DRAFTS: DraftProposal[] = [
  {
    id: 'DRF-209',
    databaseId: 'db-ledger',
    description: 'Backfill winter loyalty credits on ledger_accounts',
    proposerId: 'usr-4',
    proposerName: 'irvin',
    mode: 'batch_dml',
    status: 'pending_approval',
    upSql: `UPDATE ledger_accounts\nSET voucher_balance_cents = voucher_balance_cents + 500,\n    voucher_expires_at = '2027-01-31 23:59:59+00'\nWHERE id > $cursor_id\n  AND id <= $cursor_id + $batch_size\n  AND status = 'active';`,
    batchConfig: {
      targetTable: 'ledger_accounts',
      batchColumn: 'id',
      batchSize: 5000,
      processedRows: 0,
      totalRows: 100000,
      currentBatch: 0,
      totalBatches: 20,
    },
    consensus: [
      {
        leaderId: 'usr-3',
        leaderName: 'hannah.okafor',
        groupName: 'Core Platform',
        status: 'approved',
        votedAt: '3h ago',
      },
      {
        leaderId: 'usr-2',
        leaderName: 'leader_alice',
        groupName: 'Payments Reliability',
        status: 'pending',
      },
    ],
    updatedRelative: '38m ago',
  },
  {
    id: 'DRF-208',
    databaseId: 'db-auth',
    description: 'Add users.phone and drop legacy_field',
    proposerId: 'usr-1',
    proposerName: 'wendy',
    mode: 'normal',
    status: 'approved',
    upSql: `ALTER TABLE users\n  ADD COLUMN phone VARCHAR(20),\n  DROP COLUMN legacy_field;\n\nCREATE INDEX idx_users_phone ON users (phone);`,
    consensus: [
      {
        leaderId: 'usr-2',
        leaderName: 'leader_alice',
        groupName: 'Payments Reliability',
        status: 'approved',
        votedAt: '1d ago',
      },
      {
        leaderId: 'usr-3',
        leaderName: 'hannah.okafor',
        groupName: 'Core Platform',
        status: 'approved',
        votedAt: '1d ago',
      },
    ],
    updatedRelative: '1d ago',
  },
  {
    id: 'DRF-207',
    databaseId: 'db-analytics',
    description: 'Backfill event_partition_key on audit_stream',
    proposerId: 'usr-4',
    proposerName: 'irvin',
    mode: 'batch_dml',
    status: 'partially_reverted',
    upSql: `ALTER TABLE audit_stream ADD COLUMN event_partition_key VARCHAR(32);`,
    batchConfig: {
      targetTable: 'audit_stream',
      batchColumn: 'event_id',
      batchSize: 10000,
      processedRows: 40000,
      totalRows: 100000,
      currentBatch: 4,
      totalBatches: 10,
    },
    consensus: [
      {
        leaderId: 'usr-2',
        leaderName: 'leader_alice',
        groupName: 'Payments Reliability',
        status: 'approved',
        votedAt: '4d ago',
      },
    ],
    updatedRelative: '4d ago',
    failureMessage: 'Batch chunk 5 failed with 423 LOCKED: Deadlock detected with concurrent ETL load',
  },
];

let memoryDrafts: DraftProposal[] = [...INITIAL_DRAFTS];

export const fetchDrafts = async (): Promise<DraftProposal[]> => {
  await new Promise((resolve) => setTimeout(resolve, 100));
  return [...memoryDrafts];
};

export const fetchDraftById = async (id: string): Promise<DraftProposal | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 80));
  return memoryDrafts.find((d) => d.id === id);
};

export const fetchDraftsByDatabaseId = async (databaseId: string): Promise<DraftProposal[]> => {
  await new Promise((resolve) => setTimeout(resolve, 100));
  return memoryDrafts.filter((d) => d.databaseId === databaseId);
};

export const createDraft = async (input: {
  databaseId: string;
  title?: string;
  description: string;
  proposerId: string;
  proposerName: string;
  mode: 'normal' | 'batch_dml';
  upSql: string;
  targetTable?: string;
  batchColumn?: string;
  batchSize?: number;
  batchConfig?: BatchConfig;
  linkedGroups?: { id: string; name: string; leaderId: string; leaderName: string }[];
}): Promise<DraftProposal> => {
  await new Promise((resolve) => setTimeout(resolve, 150));
  const nextNum = 210 + memoryDrafts.length;
  const draftId = `DRF-${nextNum}`;

  const consensusVotes: LeaderConsensusVote[] = input.linkedGroups
    ? input.linkedGroups.map((g) => ({
        groupName: g.name,
        leaderId: g.leaderId,
        leaderName: g.leaderName,
        status: 'pending',
      }))
    : [
        {
          leaderId: 'usr-2',
          leaderName: 'leader_alice',
          groupName: 'Payments Reliability',
          status: 'pending',
        },
        {
          leaderId: 'usr-3',
          leaderName: 'hannah.okafor',
          groupName: 'Core Platform',
          status: 'pending',
        },
      ];

  let batchConfig = input.batchConfig;
  if (!batchConfig && input.mode === 'batch_dml') {
    const parsedSize = Number(input.batchSize) || 5000;
    const totalRows = parsedSize * 20;
    batchConfig = {
      targetTable: input.targetTable || 'target_table',
      batchColumn: input.batchColumn || 'id',
      batchSize: parsedSize,
      totalBatches: 20,
      currentBatch: 0,
      processedRows: 0,
      totalRows,
    };
  }

  const newDraft: DraftProposal = {
    id: draftId,
    databaseId: input.databaseId,
    description: input.description,
    proposerId: input.proposerId,
    proposerName: input.proposerName,
    mode: input.mode,
    status: 'pending_approval',
    updatedRelative: 'Just now',
    upSql: input.upSql,
    consensus: consensusVotes,
    batchConfig,
  };

  memoryDrafts = [newDraft, ...memoryDrafts];
  return newDraft;
};

export const voteDraft = async (
  draftId: string,
  leaderId: string,
  status: LeaderVoteStatus,
  rejectionReason?: string
): Promise<DraftProposal> => {
  await new Promise((resolve) => setTimeout(resolve, 120));
  const draft = memoryDrafts.find((d) => d.id === draftId);
  if (!draft) throw new Error('Draft not found');

  const nextConsensus = draft.consensus.map((v) => {
    if (v.leaderId !== leaderId) return v;
    return {
      ...v,
      status,
      votedAt: 'Just now',
      vetoReason: status === 'rejected' ? rejectionReason : undefined,
      reason: status === 'rejected' ? rejectionReason : undefined,
    };
  });

  const hasRejection = nextConsensus.some((v) => v.status === 'rejected');
  const allApproved = nextConsensus.every((v) => v.status === 'approved');

  let nextStatus = draft.status;
  if (hasRejection) {
    nextStatus = 'rejected';
  } else if (allApproved) {
    nextStatus = 'approved';
  }

  const updatedDraft: DraftProposal = {
    ...draft,
    consensus: nextConsensus,
    status: nextStatus,
  };

  memoryDrafts = memoryDrafts.map((d) => (d.id === draftId ? updatedDraft : d));
  return updatedDraft;
};

export const executeNormalDraft = async (
  draftId: string,
  headVersion = 1,
  executedBy = 'admin'
): Promise<{ draft: DraftProposal; migration?: MigrationRecord }> => {
  await new Promise((resolve) => setTimeout(resolve, 200));
  const draft = memoryDrafts.find((d) => d.id === draftId);
  if (!draft) throw new Error('Draft not found');

  const nextVersion = headVersion + 1;
  const appliedMig: MigrationRecord = {
    id: `mig-${Date.now()}`,
    databaseId: draft.databaseId,
    version: nextVersion,
    summary: draft.description,
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy,
    executedAtRelative: 'Just now',
    durationMs: 165,
    upSql: draft.upSql,
    diffFromVersion: headVersion,
    schemaDiff: [
      { type: 'header', content: `@@ ${draft.description} (#${headVersion} → #${nextVersion}) @@` },
      { type: 'add', content: '+ Applied draft changes successfully' },
    ],
  };

  appendMigration(appliedMig);
  updateDatabaseHeadVersion(draft.databaseId, nextVersion, []);

  const updatedDraft: DraftProposal = { ...draft, status: 'applied' };
  memoryDrafts = memoryDrafts.map((d) => (d.id === draftId ? updatedDraft : d));

  return { draft: updatedDraft, migration: appliedMig };
};

export const simulateBatchProgress = async (
  draftId: string,
  direction: 'forward' | 'rollback',
  headVersion = 1,
  executedBy = 'admin'
): Promise<{ draft: DraftProposal; appliedMigration?: MigrationRecord }> => {
  await new Promise((resolve) => setTimeout(resolve, 150));
  const draft = memoryDrafts.find((d) => d.id === draftId);
  if (!draft || !draft.batchConfig) throw new Error('Draft or batch config not found');

  const totalBatches = draft.batchConfig.totalBatches;
  const stepSize = Math.max(1, Math.ceil(totalBatches / 4));
  let appliedMigration: MigrationRecord | undefined;

  if (direction === 'forward') {
    const nextBatch = Math.min(totalBatches, draft.batchConfig.currentBatch + stepSize);
    const completed = nextBatch >= totalBatches;
    const nextRows = completed
      ? draft.batchConfig.totalRows
      : Math.round((nextBatch / totalBatches) * draft.batchConfig.totalRows);

    if (completed) {
      const nextVersion = headVersion + 1;
      appliedMigration = {
        id: `mig-${Date.now()}`,
        databaseId: draft.databaseId,
        version: nextVersion,
        summary: draft.description,
        type: 'UP',
        status: 'Success',
        reversible: true,
        executedBy,
        executedAtRelative: 'Just now',
        durationMs: 480,
        upSql: draft.upSql,
        diffFromVersion: headVersion,
        schemaDiff: [
          { type: 'header', content: `@@ ${draft.description} (#${headVersion} → #${nextVersion}) @@` },
          { type: 'add', content: `+ Batch executed ${nextRows} rows committed` },
        ],
      };
      appendMigration(appliedMigration);
      updateDatabaseHeadVersion(draft.databaseId, nextVersion, []);
      updateDatabaseQuarantine(draft.databaseId, false);
    }

    const updatedDraft: DraftProposal = {
      ...draft,
      status: completed ? 'applied' : 'executing',
      batchConfig: {
        ...draft.batchConfig,
        currentBatch: nextBatch,
        processedRows: nextRows,
      },
    };
    memoryDrafts = memoryDrafts.map((d) => (d.id === draftId ? updatedDraft : d));
    return { draft: updatedDraft, appliedMigration };
  } else {
    // Rollback batch
    const nextBatch = Math.max(0, draft.batchConfig.currentBatch - stepSize);
    const rolledBack = nextBatch === 0;
    const nextRows = Math.round((nextBatch / totalBatches) * draft.batchConfig.totalRows);

    if (rolledBack) {
      updateDatabaseQuarantine(draft.databaseId, false);
    }

    const updatedDraft: DraftProposal = {
      ...draft,
      status: rolledBack ? 'approved' : 'partially_reverted',
      batchConfig: {
        ...draft.batchConfig,
        currentBatch: nextBatch,
        processedRows: nextRows,
      },
    };
    memoryDrafts = memoryDrafts.map((d) => (d.id === draftId ? updatedDraft : d));
    return { draft: updatedDraft };
  }
};
