export type DraftStatus =
  | 'pending_approval'
  | 'approved'
  | 'applied'
  | 'executing'
  | 'partially_reverted'
  | 'rejected'
  | 'failed'
  | 'invalidated';

export type LeaderVoteStatus = 'pending' | 'approved' | 'rejected';

export interface LeaderConsensusVote {
  leaderId: string;
  leaderName: string;
  groupName: string;
  status: LeaderVoteStatus;
  votedAt?: string;
  vetoReason?: string;
  reason?: string;
}

export type ConsensusVote = LeaderConsensusVote;
export type DraftProposalMode = 'normal' | 'batch_dml';

export interface BatchConfig {
  targetTable: string;
  batchColumn: string;
  batchSize: number;
  processedRows: number;
  totalRows: number;
  totalEstimatedRows?: number;
  currentBatch: number;
  totalBatches: number;
  lastProcessedId?: number;
}

export interface DraftProposal {
  id: string;
  databaseId: string;
  description: string;
  proposerId: string;
  proposerName: string;
  mode: 'normal' | 'batch_dml';
  status: DraftStatus;
  upSql: string;
  downSql?: string;
  batchConfig?: BatchConfig;
  consensus: LeaderConsensusVote[];
  updatedRelative: string;
  failureMessage?: string;
}

export interface DraftFormInput {
  databaseId: string;
  description: string;
  mode: 'normal' | 'batch_dml';
  upSql: string;
  targetTable: string;
  batchColumn: string;
  batchSize: string;
}

export interface ValidationErrors {
  valid: boolean;
  errors: Record<string, string>;
}
