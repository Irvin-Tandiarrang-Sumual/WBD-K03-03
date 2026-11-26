import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { draftService } from '../services/draft_service';
import type { DraftProposal } from '../types';
import { useAsyncAction } from './useAsyncAction';

export function useApprovals(initialDrafts: DraftProposal[]) {
  const { user, showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [drafts, setDrafts] = useState<DraftProposal[]>(initialDrafts);
  const [databaseFilter, setDatabaseFilter] = useState('all');

  const [reviewingDraft, setReviewingDraft] = useState<DraftProposal | null>(null);
  const [vetoingDraft, setVetoingDraft] = useState<DraftProposal | null>(null);

  const pendingApprovals = useMemo(() => {
    return drafts.filter((d) => {
      const isReviewableStatus =
        d.status === 'pending_approval' ||
        d.status === 'partial' ||
        d.status === 'partially_reverted';
      if (!isReviewableStatus) return false;

      if (user.role === 'admin') return true;
      return d.consensus.some(
        (c) =>
          c.status === 'pending' &&
          (c.leaderId === user.id || c.leaderId === `leader-grp-${user.id}`)
      );
    });
  }, [drafts, user]);

  const filteredApprovals = useMemo(() => {
    if (databaseFilter === 'all') return pendingApprovals;
    return pendingApprovals.filter((d) => d.databaseId === databaseFilter);
  }, [pendingApprovals, databaseFilter]);

  const handleApprove = async (draft: DraftProposal) => {
    setReviewingDraft(null);
    const voteChoice =
      draft.status === 'partial'
        ? 'resume'
        : draft.status === 'partially_reverted'
        ? 'yes'
        : 'approved';

    await runAsync(
      async () => {
        const updated = await draftService.submitApproval(draft.id, voteChoice);
        return updated;
      },
      {
        successMessage: (updated) =>
          updated.status === 'approved'
            ? `Consensus reached! ${draft.id} is approved and ready for execution.`
            : `Vote recorded: ${voteChoice.toUpperCase()} for ${draft.id}`,
        successTone: 'pine',
        onSuccess: (updated) => {
          setDrafts((prev) => prev.map((d) => (d.id === draft.id ? updated : d)));
        },
      }
    );
  };

  const handleConfirmVeto = async (draft: DraftProposal, _reason: string) => {
    setVetoingDraft(null);
    const voteChoice =
      draft.status === 'partial'
        ? 'revert'
        : draft.status === 'partially_reverted'
        ? 'no'
        : 'rejected';

    await runAsync(
      async () => {
        const updated = await draftService.submitApproval(draft.id, voteChoice);
        return updated;
      },
      {
        successMessage: `Vote recorded: ${voteChoice.toUpperCase()} for ${draft.id}`,
        successTone: 'maroon',
        onSuccess: (updated) => {
          setDrafts((prev) => prev.map((d) => (d.id === draft.id ? updated : d)));
        },
      }
    );
  };

  return {
    drafts,
    pendingApprovals,
    filteredApprovals,
    databaseFilter,
    setDatabaseFilter,
    reviewingDraft,
    setReviewingDraft,
    vetoingDraft,
    setVetoingDraft,
    handleApprove,
    handleConfirmVeto,
  };
}
