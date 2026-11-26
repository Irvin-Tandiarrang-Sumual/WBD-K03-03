import { useState } from 'react';
import { useNavigate } from 'react-router';
import { useAuth } from '../context/auth-context';
import { draftService } from '../services/draft_service';
import { validateDraftForm } from '../utils/validation';
import type { DraftFormInput } from '../types';
import { useAsyncAction } from './useAsyncAction';

export function useCreateDraft(defaultDbId: string) {
  const navigate = useNavigate();
  const { user, showSnackbar } = useAuth();
  const { isLoading: isSubmitting, execute: runAsync } = useAsyncAction(showSnackbar);

  const [form, setForm] = useState<DraftFormInput>({
    databaseId: defaultDbId,
    description: '',
    mode: 'normal',
    upSql: '',
    targetTable: 'orders',
    batchColumn: 'id',
    batchSize: '500',
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const updateField = <K extends keyof DraftFormInput>(key: K, value: DraftFormInput[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const val = validateDraftForm(form);
    if (!val.valid) {
      setErrors(val.errors);
      return;
    }
    setErrors({});

    await runAsync(
      async () => {
        const isBatch = form.mode === 'batch_dml';
        const created = await draftService.createDraft(form.databaseId, {
          sqlUp: form.upSql.trim(),
          isBatch,
          batchColumn: isBatch ? form.batchColumn : undefined,
          batchSize: isBatch ? parseInt(form.batchSize, 10) || 500 : undefined,
        });
        return created;
      },
      {
        successMessage: (created) => `Draft ${created.id} submitted for leader consensus!`,
        successTone: 'pine',
        errorMessage: 'Failed to submit migration draft',
        onSuccess: (created) => {
          navigate(`/drafts/${created.id}`);
        },
      }
    );
  };

  return {
    form,
    errors,
    isSubmitting,
    updateField,
    handleSubmit,
  };
}
