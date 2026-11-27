import React from 'react';
import { useLoaderData, useNavigate } from 'react-router';
import {
  CheckCircle2,
  Clock,
  Layers,
  Play,
  RefreshCw,
  RotateCcw,
  Shield,
  XCircle,
} from 'lucide-react';
import { SqlCodeViewer } from '../components/code/SqlCodeViewer';
import { PageHeader } from '../components/common/PageHeader';
import { useDraftDetail } from '../hooks/useDraftDetail';
import { databaseService } from '../services/database_service';
import { draftService } from '../services/draft_service';
import type { ConsensusVote, DraftProposal, DraftStatus, TargetDatabase } from '../types';

export async function clientLoader({ params }: { params: { id?: string } }) {
  const draftId = params.id || '1';
  let draft: DraftProposal | null = null;
  try {
    draft = await draftService.getDraftById(draftId);
  } catch {
    throw new Response('Draft Proposal Not Found', { status: 404 });
  }

  const databases = await databaseService.getAccessibleDatabases();
  const allDrafts = await draftService.getAllDraftsAcrossDatabases(databases);

  return { draft, databases, allDrafts };
}

export default function DraftDetailPage() {
  const { draft: initialDraft, databases, allDrafts } = useLoaderData() as {
    draft: DraftProposal;
    databases: TargetDatabase[];
    allDrafts: DraftProposal[];
  };

  const navigate = useNavigate();
  const d = useDraftDetail({ initialDraft, databases });

  return (
    <section className="space-y-5">
      <PageHeader
        title={`${d.draft.id} · ${d.draft.description}`}
        subtitle={`Target Database: ${d.targetDb?.name ?? d.draft.databaseId} · Author: ${d.draft.proposerName}`}
        badge={{
          label: d.draft.status.toUpperCase(),
          variant: d.draft.status === 'applied' ? 'success' : d.draft.status === 'failed' ? 'danger' : 'warning',
        }}
        actions={
          <select
            aria-label="Switch to another draft"
            value={d.draft.id}
            onChange={(e) => navigate(`/drafts/${e.target.value}`)}
            className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            {allDrafts.map((drf) => (
              <option key={drf.id} value={drf.id}>
                {drf.id}: {drf.description}
              </option>
            ))}
          </select>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-5">
          <BatchExecutionCard
            draft={d.draft}
            progressPercent={d.progressPercent}
            isExecuting={d.isExecuting}
            onBatchStep={d.handleBatchStep}
            onVote={d.handleVote}
          />

          {d.showNormalExecuteBar && (
            <ExecutionStatusBar
              isExecuting={d.isExecuting}
              onExecute={d.handleExecuteNormal}
              approvedCount={d.draft.consensus.filter((c) => c.status === 'approved').length}
              totalCount={d.draft.consensus.length}
              canExecute={d.canExecute}
              proposerName={d.draft.proposerName}
            />
          )}

          <ExecutionErrorAlert message={d.draft.failureMessage} />

          <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs p-5 space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-bold text-[#3A241A]">{d.draft.description}</h2>
              <span className="font-mono text-[11px] text-[#6B5444]">
                Up SQL · Rollback system-handled
              </span>
            </div>
            <SqlCodeViewer code={d.draft.upSql} maxHeight="max-h-[300px]" />
          </div>
        </div>

        <div className="space-y-5">
          <ConsensusQuorumPanel
            votes={d.draft.consensus}
            draftStatus={d.draft.status}
            myPendingVote={d.myPendingVote}
            onVote={d.handleVote}
          />
        </div>
      </div>
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface BatchExecutionCardProps {
  draft: DraftProposal;
  progressPercent: number;
  isExecuting: boolean;
  onBatchStep: (direction: 'forward' | 'rollback') => void;
  onVote: (status: string) => void;
}

const BatchExecutionCard: React.FC<BatchExecutionCardProps> = ({
  draft,
  progressPercent,
  isExecuting,
  onBatchStep,
  onVote,
}) => {
  if (!draft.batchConfig) return null;

  const { targetTable, batchColumn, currentBatch, totalBatches, processedRows, totalRows } =
    draft.batchConfig;
  const isPartiallyReverted = draft.status === 'partially_reverted';
  const isApprovedOrExecuting = draft.status === 'approved' || draft.status === 'executing';
  const actionButtonLabel = draft.status === 'approved' ? 'Execute Batch' : 'Advance Batch';

  return (
    <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
      <div className="px-5 py-3.5 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs font-bold text-[#3A241A]">
          <Layers className="w-4 h-4 text-[#264D3B]" />
          <span>Batch Chunking Execution</span>
          <span className="font-mono font-normal text-[#6B5444]">
            {targetTable} ({batchColumn})
          </span>
        </div>
        <span className="font-mono text-xs font-bold text-[#264D3B]">
          Batch {currentBatch}/{totalBatches} · {processedRows.toLocaleString()} /{' '}
          {totalRows.toLocaleString()} rows ({progressPercent}%)
        </span>
      </div>

      <div className="p-5 space-y-4">
        <div className="relative w-full h-4 rounded-full bg-[#EFE5D1] border border-[#D8C7A5] overflow-hidden">
          <svg className="w-full h-full">
            <defs>
              <linearGradient id="batchGradient" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="#264D3B" />
                <stop offset="60%" stopColor="#4C7A5B" />
                <stop offset="100%" stopColor="#F3B738" />
              </linearGradient>
            </defs>
            <rect
              x="0"
              y="0"
              width={`${Math.max(progressPercent, 4)}%`}
              height="100%"
              rx="8"
              fill="url(#batchGradient)"
              className="transition-all duration-500"
            />
          </svg>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
          <div className="text-xs text-[#6B5444]">
            {draft.status === 'applied' && (
              <span className="text-[#264D3B] font-bold">
                All batches executed successfully.
              </span>
            )}
            {draft.status === 'executing' && (
              <span>Chunk execution in progress. Advance batch to proceed.</span>
            )}
            {draft.status === 'failed' && (
              <span className="text-[#822433] font-bold">
                Batch failure occurred. Rollback available.
              </span>
            )}
            {draft.status === 'partial' && (
              <span className="text-[#8A5A00] font-bold">
                {draft.recoveryAction === 'resume'
                  ? 'Consensus reached: Resume. Click Resume Batch to continue execution.'
                  : draft.recoveryAction === 'revert'
                  ? 'Consensus reached: Revert. Click Rollback Step to revert committed batches.'
                  : 'Batch execution paused on error. Awaiting leader consensus vote (Resume vs Revert).'}
              </span>
            )}
            {isPartiallyReverted && (
              <span className="text-[#822433] font-bold">
                {draft.recoveryAction === 'revert'
                  ? 'Consensus reached: Continue Revert. Click Rollback Step to retry.'
                  : draft.recoveryAction === 'aborted'
                  ? 'Rollback aborted by leader vote. Database left in intermediate state.'
                  : 'Rollback failed midway. Awaiting leader consensus vote (Continue Revert: Yes vs No).'}
              </span>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {draft.status === 'partial' && draft.recoveryAction === 'revert' && (
              <button
                type="button"
                disabled={isExecuting}
                onClick={() => onVote('resume')}
                className="h-9 px-3.5 rounded-xl border border-[#264D3B]/40 bg-[#264D3B]/10 hover:bg-[#264D3B] hover:text-[#F5EEDD] text-[#264D3B] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
              >
                <Play className="w-3.5 h-3.5" />
                <span>Switch to Resume</span>
              </button>
            )}

            {draft.status === 'partial' && draft.recoveryAction === 'resume' && (
              <button
                type="button"
                disabled={isExecuting}
                onClick={() => onVote('revert')}
                className="h-9 px-3.5 rounded-xl border border-[#822433]/35 bg-[#FFFDF9] hover:bg-[#822433]/10 text-[#822433] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
              >
                <RotateCcw className="w-3.5 h-3.5" />
                <span>Switch to Revert</span>
              </button>
            )}

            {(draft.status === 'failed' ||
              (draft.status === 'partial' && draft.recoveryAction === 'revert') ||
              (isPartiallyReverted && draft.recoveryAction === 'revert')) && (
              <button
                type="button"
                disabled={isExecuting || currentBatch === 0}
                onClick={() => onBatchStep('rollback')}
                className="h-9 px-4 rounded-xl border border-[#822433]/40 bg-[#822433]/10 hover:bg-[#822433] hover:text-white text-[#822433] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
              >
                <RotateCcw className="w-3.5 h-3.5" />
                <span>Rollback Step</span>
              </button>
            )}

            {(isApprovedOrExecuting ||
              (draft.status === 'partial' && draft.recoveryAction === 'resume')) && (
              <button
                type="button"
                disabled={isExecuting || currentBatch >= totalBatches}
                onClick={() => onBatchStep('forward')}
                className="h-9 px-4 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] disabled:opacity-50 text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
              >
                {isExecuting ? (
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Play className="w-3.5 h-3.5" />
                )}
                <span>{draft.status === 'partial' ? 'Resume Batch' : actionButtonLabel}</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

const ConsensusQuorumPanel: React.FC<{
  votes: ConsensusVote[];
  draftStatus: DraftStatus;
  myPendingVote?: ConsensusVote;
  onVote: (status: string) => void;
}> = ({ votes, draftStatus, myPendingVote, onVote }) => {
  const approvedCount = votes.filter(
    (c) => c.status === 'approved' || c.status === 'resume' || c.status === 'yes'
  ).length;

  return (
    <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs p-5 space-y-4">
      <div className="flex items-center justify-between border-b border-[#EFE5D1] pb-3">
        <span className="text-xs font-bold text-[#3A241A] inline-flex items-center gap-1.5">
          <Shield className="w-4 h-4 text-[#264D3B]" />
          <span>Consensus Quorum</span>
        </span>
        <span className="font-mono text-xs font-bold text-[#264D3B]">
          {approvedCount}/{votes.length} Decided
        </span>
      </div>

      <div className="space-y-3">
        {votes.map((vote) => (
          <div
            key={vote.leaderId}
            className="rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/50 p-3.5 space-y-1.5"
          >
            <div className="flex items-center justify-between">
              <span className="font-bold text-xs text-[#3A241A]">{vote.leaderName}</span>
              <span
                className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold ${getVoteBadgeStyle(
                  vote.status
                )}`}
              >
                <VoteStatusIcon status={vote.status} />
                <span>{vote.status.toUpperCase()}</span>
              </span>
            </div>
            <div className="text-[11px] font-mono text-[#6B5444]">Group: {vote.groupName}</div>
            {(vote.vetoReason || vote.reason) && (
              <div className="text-[11px] text-[#822433] bg-[#822433]/8 p-2 rounded-lg border border-[#822433]/20">
                {vote.vetoReason || vote.reason}
              </div>
            )}
          </div>
        ))}
      </div>

      {myPendingVote && (
        <div className="pt-3 border-t border-[#EFE5D1] space-y-2">
          <div className="text-xs font-semibold text-[#3A241A]">
            {draftStatus === 'partial'
              ? 'Recovery voting requested (Resume execution vs Rollback batch):'
              : draftStatus === 'partially_reverted'
              ? 'Rollback failure decision (Continue revert: Yes vs No):'
              : 'Your leader consensus vote is requested:'}
          </div>
          <div className="flex items-center gap-2">
            {draftStatus === 'partial' ? (
              <>
                <button
                  type="button"
                  onClick={() => onVote('resume')}
                  className="flex-1 h-9 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-white text-xs font-semibold inline-flex items-center justify-center gap-1 shadow-xs cursor-pointer transition-colors"
                >
                  <Play className="w-3.5 h-3.5" />
                  <span>Resume</span>
                </button>
                <button
                  type="button"
                  onClick={() => onVote('revert')}
                  className="flex-1 h-9 rounded-xl border border-[#822433] text-[#822433] hover:bg-[#822433]/10 text-xs font-semibold inline-flex items-center justify-center gap-1 cursor-pointer transition-colors"
                >
                  <RotateCcw className="w-3.5 h-3.5" />
                  <span>Revert</span>
                </button>
              </>
            ) : draftStatus === 'partially_reverted' ? (
              <>
                <button
                  type="button"
                  onClick={() => onVote('yes')}
                  className="flex-1 h-9 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-white text-xs font-semibold inline-flex items-center justify-center gap-1 shadow-xs cursor-pointer transition-colors"
                >
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>Yes (Lanjut)</span>
                </button>
                <button
                  type="button"
                  onClick={() => onVote('no')}
                  className="flex-1 h-9 rounded-xl border border-[#822433] text-[#822433] hover:bg-[#822433]/10 text-xs font-semibold inline-flex items-center justify-center gap-1 cursor-pointer transition-colors"
                >
                  <XCircle className="w-3.5 h-3.5" />
                  <span>No (Batal)</span>
                </button>
              </>
            ) : (
              <>
                <button
                  type="button"
                  onClick={() => onVote('approved')}
                  className="flex-1 h-9 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-white text-xs font-semibold inline-flex items-center justify-center gap-1 shadow-xs cursor-pointer transition-colors"
                >
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>Approve</span>
                </button>
                <button
                  type="button"
                  onClick={() => onVote('rejected')}
                  className="flex-1 h-9 rounded-xl border border-[#822433] text-[#822433] hover:bg-[#822433]/10 text-xs font-semibold inline-flex items-center justify-center gap-1 cursor-pointer transition-colors"
                >
                  <XCircle className="w-3.5 h-3.5" />
                  <span>Reject</span>
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

const ExecutionStatusBar: React.FC<{
  isExecuting: boolean;
  onExecute: () => void;
  approvedCount: number;
  totalCount: number;
  canExecute: boolean;
  proposerName: string;
}> = ({ isExecuting, onExecute, approvedCount, totalCount, canExecute, proposerName }) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs px-5 py-4 flex flex-wrap items-center justify-between gap-3">
    <span className="text-xs font-bold text-[#264D3B]">
      Consensus quorum reached ({approvedCount}/{totalCount}) · {canExecute ? 'Ready to execute on target database' : `Awaiting author (${proposerName}) to execute`}
    </span>
    {canExecute && (
      <button
        type="button"
        disabled={isExecuting}
        onClick={onExecute}
        className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
      >
        <Play className="w-4 h-4" />
        <span>Execute Migration</span>
      </button>
    )}
  </div>
);

const ExecutionErrorAlert: React.FC<{ message?: string }> = ({ message }) => {
  if (!message) return null;

  return (
    <div className="rounded-2xl bg-[#822433] text-white p-4 font-mono text-xs leading-relaxed shadow-md">
      <div className="flex items-center gap-2 font-sans font-bold text-xs mb-1.5 text-[#F3B738]">
        <XCircle className="w-4 h-4" />
        <span>Execution Error (423 Quarantined)</span>
      </div>
      <p>{message}</p>
    </div>
  );
};

// --- Low-Level Helpers (Stepdown Rule) ---

function getVoteBadgeStyle(status: ConsensusVote['status']): string {
  switch (status) {
    case 'approved':
    case 'resume':
    case 'yes':
      return 'bg-[#264D3B]/15 text-[#264D3B]';
    case 'rejected':
    case 'no':
      return 'bg-[#822433]/15 text-[#822433]';
    case 'revert':
      return 'bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70';
    case 'pending':
    default:
      return 'bg-[#F3B738]/25 text-[#6E4900]';
  }
}

function VoteStatusIcon({ status }: { status: ConsensusVote['status'] }) {
  if (status === 'approved' || status === 'resume' || status === 'yes') {
    return <CheckCircle2 className="w-3 h-3" />;
  }
  if (status === 'rejected' || status === 'no') {
    return <XCircle className="w-3 h-3" />;
  }
  if (status === 'revert') {
    return <RotateCcw className="w-3 h-3" />;
  }
  return <Clock className="w-3 h-3" />;
}
