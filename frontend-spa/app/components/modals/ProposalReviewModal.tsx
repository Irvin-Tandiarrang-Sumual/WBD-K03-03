import React from 'react';
import { CheckCircle2, FileCode2, XCircle } from 'lucide-react';
import type { ConsensusVote, DraftProposal, TargetDatabase } from '../../types';
import { M3Dialog } from '../common/M3Dialog';
import { SqlCodeViewer } from '../code/SqlCodeViewer';

export interface ProposalReviewModalProps {
  draft: DraftProposal | null;
  database?: TargetDatabase;
  onClose: () => void;
  onApprove: (draft: DraftProposal) => void;
  onReject: (draft: DraftProposal) => void;
}

const ProposalMetadataGrid = ({
  draft,
  database,
}: {
  draft: DraftProposal;
  database?: TargetDatabase;
}) => (
  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <div className="rounded-xl bg-[#F5EEDD]/70 border border-[#D8C7A5] p-3">
      <p className="text-[10px] font-bold uppercase tracking-wider text-[#6B5444]">
        Target Database
      </p>
      <p className="mt-1 font-mono text-xs font-bold text-[#3A241A] truncate">
        {database?.name ?? draft.databaseId}
      </p>
    </div>
    <div className="rounded-xl bg-[#F5EEDD]/70 border border-[#D8C7A5] p-3">
      <p className="text-[10px] font-bold uppercase tracking-wider text-[#6B5444]">
        Proposer
      </p>
      <p className="mt-1 text-xs font-bold text-[#3A241A] truncate">{draft.proposerName}</p>
    </div>
    <div className="rounded-xl bg-[#F5EEDD]/70 border border-[#D8C7A5] p-3">
      <p className="text-[10px] font-bold uppercase tracking-wider text-[#6B5444]">
        Execution Mode
      </p>
      <p className="mt-1 text-xs font-bold text-[#1E6B8C]">
        {draft.mode === 'batch_dml' ? 'Batch DML' : 'Standard DDL'}
      </p>
    </div>
    <div className="rounded-xl bg-[#F5EEDD]/70 border border-[#D8C7A5] p-3">
      <p className="text-[10px] font-bold uppercase tracking-wider text-[#6B5444]">
        Submitted
      </p>
      <p className="mt-1 font-mono text-xs font-bold text-[#3A241A]">
        {draft.updatedRelative}
      </p>
    </div>
  </div>
);

const getQuorumVoteBadgeStyle = (status: ConsensusVote['status']) => {
  switch (status) {
    case 'approved':
      return 'bg-[#264D3B]/12 text-[#264D3B] border-[#264D3B]/30';
    case 'rejected':
      return 'bg-[#822433]/12 text-[#822433] border-[#822433]/30';
    case 'pending':
    default:
      return 'bg-[#F3B738]/20 text-[#6E4900] border-[#F3B738]/60';
  }
};

const ConsensusBreakdown = ({ votes }: { votes: ConsensusVote[] }) => {
  const approvedCount = votes.filter((v) => v.status === 'approved').length;

  return (
    <div className="rounded-xl bg-[#F5EEDD]/55 border border-[#D8C7A5] p-3.5 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] font-bold uppercase tracking-wider text-[#6B5444]">
          Leader Consensus Quorum
        </span>
        <span className="text-[11px] font-mono font-semibold text-[#3A241A]">
          {approvedCount}/{votes.length} approved
        </span>
      </div>
      <div className="flex flex-wrap gap-2">
        {votes.map((vote) => (
          <span
            key={vote.leaderId}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold border ${getQuorumVoteBadgeStyle(
              vote.status
            )}`}
          >
            <span>{vote.leaderName}</span>
            <span className="opacity-70">({vote.groupName})</span>
            <span className="uppercase text-[10px] font-bold">· {vote.status}</span>
          </span>
        ))}
      </div>
    </div>
  );
};

export const ProposalReviewModal: React.FC<ProposalReviewModalProps> = ({
  draft,
  database,
  onClose,
  onApprove,
  onReject,
}) => {
  if (!draft) return null;

  return (
    <M3Dialog
      open={!!draft}
      onClose={onClose}
      title={`${draft.id} · Proposal Review`}
      subtitle={draft.description}
      maxWidth="max-w-2xl"
      footer={
        <>
          <button
            type="button"
            onClick={onClose}
            className="h-10 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold cursor-pointer"
          >
            Close
          </button>
          <button
            type="button"
            onClick={() => onReject(draft)}
            className="h-10 px-4 rounded-xl bg-[#822433] hover:bg-[#691B28] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            <XCircle className="w-4 h-4" />
            <span>Reject / Veto</span>
          </button>
          <button
            type="button"
            onClick={() => onApprove(draft)}
            className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            <CheckCircle2 className="w-4 h-4" />
            <span>Approve</span>
          </button>
        </>
      }
    >
      <div className="space-y-4">
        <ProposalMetadataGrid draft={draft} database={database} />

        <ConsensusBreakdown votes={draft.consensus} />

        {draft.batchConfig && (
          <div className="rounded-xl bg-[#1E6B8C]/10 border border-[#1E6B8C]/30 px-3.5 py-2.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[#3A241A]">
            <span className="font-bold text-[#1E6B8C]">Batch DML Chunking:</span>
            <span className="font-mono">table={draft.batchConfig.targetTable}</span>
            <span className="font-mono">key={draft.batchConfig.batchColumn}</span>
            <span className="font-mono">
              size={draft.batchConfig.batchSize.toLocaleString()} rows
            </span>
          </div>
        )}

        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold text-[#3A241A] inline-flex items-center gap-1.5">
              <FileCode2 className="w-4 h-4 text-[#1E6B8C]" />
              <span>Up Migration SQL Script</span>
            </span>
            <span className="text-[11px] font-mono text-[#6B5444]">Read-only preview</span>
          </div>
          <SqlCodeViewer code={draft.upSql} maxHeight="max-h-[260px]" />
        </div>
      </div>
    </M3Dialog>
  );
};
