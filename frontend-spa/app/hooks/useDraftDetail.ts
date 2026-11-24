import { useState } from 'react';
import { useAuth } from '../context/auth-context';
import {
  updateDatabaseHeadVersion,
  updateDatabaseQuarantine,
} from '../services/dummy_databases';
import {
  executeNormalDraft,
  simulateBatchProgress,
} from '../services/dummy_drafts';
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

  const handleExecuteNormal = async () => {
    await runAsync(
      async () => {
        const res = await executeNormalDraft(draft.id, targetDb?.headVersion, user.username);
        return res;
      },
      {
        successMessage: `Draft ${draft.id} executed successfully and applied!`,
        successTone: 'pine',
        errorMessage: 'Execution failed',
        onSuccess: async (res) => {
          setDraft(res.draft);
          if (targetDb) {
            const nextVersion = targetDb.headVersion + 1;
            await updateDatabaseHeadVersion(targetDb.id, nextVersion, []);
          }
        },
      }
    );
  };

  const handleBatchStep = async (direction: 'forward' | 'rollback') => {
    await runAsync(
      async () => {
        const res = await simulateBatchProgress(
          draft.id,
          direction,
          targetDb?.headVersion,
          user.username
        );
        return res;
      },
      {
        errorMessage: 'Batch step failed',
        onSuccess: async (res) => {
          setDraft(res.draft);

          if (res.draft.status === 'partially_reverted') {
            if (targetDb) {
              await updateDatabaseQuarantine(targetDb.id, true, draft.id);
            }
            showSnackbar(`Batch encountered error! Locked in State 423 Quarantined.`, 'maroon');
          } else if (res.draft.status === 'applied') {
            if (targetDb) {
              await updateDatabaseQuarantine(targetDb.id, false);
            }
            showSnackbar(`Batch execution completed 100% and applied!`, 'pine');
          } else {
            showSnackbar(
              `Batch advanced: chunk ${res.draft.batchConfig?.currentBatch}/${res.draft.batchConfig?.totalBatches}`,
              'blue'
            );
          }
        },
      }
    );
  };

  const showNormalExecuteBar = !draft.batchConfig && draft.status === 'approved';

  return {
    draft,
    targetDb,
    progressPercent,
    isExecuting,
    showNormalExecuteBar,
    handleExecuteNormal,
    handleBatchStep,
  };
}
