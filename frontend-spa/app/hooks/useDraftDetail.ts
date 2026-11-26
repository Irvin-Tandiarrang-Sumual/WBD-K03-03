import { useState, useEffect } from 'react';
import { useAuth } from '../context/auth-context';
import { draftService } from '../services/draft_service';
import type { DraftProposal, TargetDatabase } from '../types';
import { useAsyncAction } from './useAsyncAction';

export interface UseDraftDetailProps {
  initialDraft: DraftProposal;
  databases: TargetDatabase[];
}

export function useDraftDetail({ initialDraft, databases }: UseDraftDetailProps) {
  const [draft, setDraft] = useState<DraftProposal>(initialDraft);
  const { user, showSnackbar } = useAuth();
  const { isLoading: isExecuting, execute: runAsync } = useAsyncAction(showSnackbar);

  const targetDb = databases.find((d) => d.id === draft.databaseId);
  const progressPercent = draft.batchConfig
    ? Math.round(
        (draft.batchConfig.processedRows / Math.max(1, draft.batchConfig.totalRows)) * 100
      )
    : 0;

  const isAuthor = user.id === draft.proposerId;
  const isAdmin = user.role === 'admin';

  useEffect(() => {
    if (!draft.batchConfig || draft.status !== 'executing') {
      return;
    }

    const unsubscribe = draftService.streamBatchProgress(
      draft.id,
      (progress) => {
        setDraft((prev) => {
          if (!prev.batchConfig) return prev;
          return {
            ...prev,
            status: (progress.status || prev.status) as any,
            batchConfig: {
              ...prev.batchConfig,
              currentBatch: progress.current_batch,
              totalBatches: progress.total_batches,
              processedRows: progress.rows_processed,
              totalRows: progress.total_rows,
            },
          };
        });
      },
      async () => {
        try {
          const refreshed = await draftService.getDraftById(draft.id);
          setDraft(refreshed);
        } catch (err) {
          console.error('[SSE] Failed to refresh draft after stream completion', err);
        }
      }
    );

    return () => {
      unsubscribe();
    };
  }, [draft.id, draft.status, !!draft.batchConfig]);

  const myPendingVote = draft.consensus.find(
    (c) =>
      c.status === 'pending' &&
      (c.leaderId === user.id || isAdmin)
  );

  const handleVote = async (status: string) => {
    await runAsync(
      async () => {
        const updated = await draftService.submitApproval(draft.id, status);
        return updated;
      },
      {
        successMessage: `Vote recorded: ${status.toUpperCase()} for draft ${draft.id}`,
        successTone: ['approved', 'resume', 'yes'].includes(status) ? 'pine' : 'maroon',
        onSuccess: (updated) => {
          setDraft(updated);
        },
      }
    );
  };

  const handleExecuteNormal = async () => {
    await runAsync(
      async () => {
        await draftService.executeDraft(draft.id);
        const updated = await draftService.getDraftById(draft.id);
        return updated;
      },
      {
        successMessage: `Draft ${draft.id} execution initiated successfully!`,
        successTone: 'pine',
        errorMessage: 'Execution failed or requires consensus approval',
        onSuccess: (updated) => {
          setDraft(updated);
        },
      }
    );
  };

  const handleBatchStep = async (direction: 'forward' | 'rollback') => {
    await runAsync(
      async () => {
        if (direction === 'forward') {
          if (draft.status === 'approved') {
            await draftService.executeDraft(draft.id);
          } else {
            await draftService.resumeBatch(draft.id);
          }
        } else {
          await draftService.revertBatch(draft.id);
        }

        const updated = await draftService.getDraftById(draft.id);
        return updated;
      },
      {
        successMessage:
          direction === 'forward'
            ? `Batch forward operation submitted`
            : `Batch rollback operation submitted`,
        successTone: direction === 'forward' ? 'pine' : 'maroon',
        errorMessage: `Batch ${direction} operation failed`,
        onSuccess: (updated) => {
          setDraft(updated);
        },
      }
    );
  };

  const showNormalExecuteBar = draft.mode === 'normal' && draft.status === 'approved';
  const canExecute = isAuthor || isAdmin;

  return {
    draft,
    targetDb,
    progressPercent,
    isExecuting,
    isAuthor,
    canExecute,
    myPendingVote,
    showNormalExecuteBar,
    handleVote,
    handleExecuteNormal,
    handleBatchStep,
  };
}
