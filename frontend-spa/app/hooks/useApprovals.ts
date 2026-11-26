import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { voteDraft } from '../services/dummy_drafts';
import type { DraftProposal } from '../types';
import { useAsyncAction } from './useAsyncAction';

export function useApprovals(initialDrafts: DraftProposal[]) {
  const { user, role, showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [drafts, setDrafts] = useState<DraftProposal[]>(initialDrafts);
  const [databaseFilter, setDatabaseFilter] = useState('all');

  const [reviewingDraft, setReviewingDraft] = useState<DraftProposal | null>(null);
  const [vetoingDraft, setVetoingDraft] = useState<DraftProposal | null>(null);

  const leaderId = role === 'Leader' ? user.id : 'usr-alice-leader';

  const pendingApprovals = useMemo(() => {
    return drafts.filter(
      (d) =>
        d.status === 'pending_approval' &&
        d.consensus.some((c) => c.leaderId === leaderId && c.status === 'pending')
    );
  }, [drafts, leaderId]);

  const filteredApprovals = useMemo(() => {
    if (databaseFilter === 'all') return pendingApprovals;
    return pendingApprovals.filter((d) => d.databaseId === databaseFilter);
  }, [pendingApprovals, databaseFilter]);

  const handleApprove = async (draft: DraftProposal) => {
    setReviewingDraft(null);
    await runAsync(
      async () => {
        const updated = await voteDraft(draft.id, leaderId, 'approved');
        return updated;
      },
      {
        successMessage: (updated) =>
          updated.status === 'approved'
            ? `Consensus reached! ${draft.id} is approved and ready for execution.`
            : `Vote recorded: Approved ${draft.id}`,
        successTone: 'pine',
        onSuccess: (updated) => {
          setDrafts((prev) => prev.map((d) => (d.id === draft.id ? updated : d)));
        },
      }
    );
  };

  const handleConfirmVeto = async (draft: DraftProposal, reason: string) => {
    setVetoingDraft(null);
    await runAsync(
      async () => {
        const updated = await voteDraft(draft.id, leaderId, 'rejected', reason);
        return updated;
      },
      {
        successMessage: `Proposal ${draft.id} rejected / vetoed.`,
        successTone: 'maroon',
        onSuccess: (updated) => {
          setDrafts((prev) => prev.map((d) => (d.id === draft.id ? updated : d)));
        },
      }
    );
  };

  return {
    databaseFilter,
    setDatabaseFilter,
    reviewingDraft,
    setReviewingDraft,
    vetoingDraft,
    setVetoingDraft,
    pendingApprovals,
    filteredApprovals,
    handleApprove,
    handleConfirmVeto,
  };
}
