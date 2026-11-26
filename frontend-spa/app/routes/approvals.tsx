import React from 'react';
import { useLoaderData } from 'react-router';
import { CheckCircle2, Eye, XCircle } from 'lucide-react';
import { EmptyState } from '../components/common/EmptyState';
import { PageHeader } from '../components/common/PageHeader';
import { ProposalReviewModal } from '../components/modals/ProposalReviewModal';
import { VetoReasonModal } from '../components/modals/VetoReasonModal';
import { useApprovals } from '../hooks/useApprovals';
import { fetchDatabases } from '../services/dummy_databases';
import { fetchDrafts } from '../services/dummy_drafts';
import type { DraftProposal, TargetDatabase } from '../types';

export async function clientLoader() {
  const [drafts, databases] = await Promise.all([fetchDrafts(), fetchDatabases()]);
  return { drafts, databases };
}

export default function ApprovalsPage() {
  const { drafts: initialDrafts, databases } = useLoaderData() as {
    drafts: DraftProposal[];
    databases: TargetDatabase[];
  };

  const d = useApprovals(initialDrafts);

  return (
    <section className="space-y-5">
      <PageHeader
        title="Leader Approval Center"
        badge={{ label: `${d.pendingApprovals.length} pending`, variant: 'warning' }}
        actions={
          <select
            aria-label="Filter approvals by target database"
            value={d.databaseFilter}
            onChange={(e) => d.setDatabaseFilter(e.target.value)}
            className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            <option value="all">Target DB: All</option>
            {databases.map((db) => (
              <option key={db.id} value={db.id}>
                {db.name}
              </option>
            ))}
          </select>
        }
      />

      <ApprovalCardsGrid
        approvals={d.filteredApprovals}
        databases={databases}
        onViewDetails={(draft) => d.setReviewingDraft(draft)}
        onReject={(draft) => d.setVetoingDraft(draft)}
        onApprove={d.handleApprove}
      />

      <ProposalReviewModal
        draft={d.reviewingDraft}
        database={databases.find((db) => db.id === d.reviewingDraft?.databaseId)}
        onClose={() => d.setReviewingDraft(null)}
        onApprove={d.handleApprove}
        onReject={(draft) => {
          d.setReviewingDraft(null);
          d.setVetoingDraft(draft);
        }}
      />

      <VetoReasonModal
        draft={d.vetoingDraft}
        onClose={() => d.setVetoingDraft(null)}
        onConfirmVeto={d.handleConfirmVeto}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface ApprovalCardsGridProps {
  approvals: DraftProposal[];
  databases: TargetDatabase[];
  onViewDetails: (draft: DraftProposal) => void;
  onReject: (draft: DraftProposal) => void;
  onApprove: (draft: DraftProposal) => void;
}

const ApprovalCardsGrid: React.FC<ApprovalCardsGridProps> = ({
  approvals,
  databases,
  onViewDetails,
  onReject,
  onApprove,
}) => {
  if (approvals.length === 0) {
    return (
      <EmptyState
        title="No Pending Approvals"
        description="No schema drafts currently require your consensus review."
      />
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {approvals.map((draft) => {
        const db = databases.find((d) => d.id === draft.databaseId);
        return (
          <ApprovalCard
            key={draft.id}
            draft={draft}
            database={db}
            onViewDetails={onViewDetails}
            onReject={onReject}
            onApprove={onApprove}
          />
        );
      })}
    </div>
  );
};

interface ApprovalCardProps {
  draft: DraftProposal;
  database?: TargetDatabase;
  onViewDetails: (draft: DraftProposal) => void;
  onReject: (draft: DraftProposal) => void;
  onApprove: (draft: DraftProposal) => void;
}

const ApprovalCard: React.FC<ApprovalCardProps> = ({
  draft,
  database,
  onViewDetails,
  onReject,
  onApprove,
}) => (
  <article
    data-testid="approval-card"
    className="min-h-[180px] flex flex-col justify-between p-6 bg-[#FFFDF9]/92 backdrop-blur-md border border-[#D8C7A5] rounded-2xl shadow-sm hover:shadow-md transition"
  >
    <div className="flex items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-2 min-w-0">
        <button
          type="button"
          onClick={() => onViewDetails(draft)}
          className="font-mono text-xs font-bold text-[#1E6B8C] hover:underline cursor-pointer shrink-0"
        >
          {draft.id}
        </button>
        <span className="px-2 py-0.5 rounded-lg bg-[#EFE5D1]/85 border border-[#D8C7A5] font-mono text-[11px] font-semibold text-[#3A241A] truncate max-w-[170px]">
          {database?.name ?? draft.databaseId}
        </span>
        <span className="text-xs text-[#6B5444] truncate">by {draft.proposerName}</span>
      </div>
      <span className="px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70 shrink-0">
        Pending
      </span>
    </div>

    <h2 className="text-[15px] font-bold text-[#3A241A] leading-snug line-clamp-2 my-3">
      {draft.description}
    </h2>

    <div className="pt-3 border-t border-[#EFE5D1] flex flex-wrap items-center justify-between gap-2">
      <span className="text-[11px] font-mono text-[#6B5444] shrink-0">
        {draft.updatedRelative}
      </span>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          aria-label={`View Details for ${draft.id}`}
          onClick={() => onViewDetails(draft)}
          className="h-9 px-3 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
        >
          <Eye className="w-3.5 h-3.5 text-[#1E6B8C]" />
          <span>View Details</span>
        </button>
        <button
          type="button"
          aria-label={`Reject or Veto ${draft.id}`}
          onClick={() => onReject(draft)}
          className="h-9 px-3 rounded-xl bg-[#822433] hover:bg-[#691B28] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer"
        >
          <XCircle className="w-3.5 h-3.5" />
          <span>Reject / Veto</span>
        </button>
        <button
          type="button"
          aria-label={`Approve ${draft.id}`}
          onClick={() => onApprove(draft)}
          className="h-9 px-3.5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer"
        >
          <CheckCircle2 className="w-3.5 h-3.5" />
          <span>Approve</span>
        </button>
      </div>
    </div>
  </article>
);
