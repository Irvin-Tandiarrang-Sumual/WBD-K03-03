import type { DraftFormInput } from '../types';

export interface ValidationResult {
  valid: boolean;
  errors: Record<string, string>;
}

export function validateDraftForm(input: DraftFormInput): ValidationResult {
  const errors: Record<string, string> = {};

  if (!input.databaseId) {
    errors.databaseId = 'Select a target database.';
  }

  if (!input.description || input.description.trim().length < 3) {
    errors.description = 'Proposal summary / title is required.';
  }

  if (!input.upSql.trim()) {
    errors.upSql = 'Up Migration SQL is required.';
  }

  if (input.mode === 'batch_dml') {
    validateBatchParameters(input, errors);
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  };
}

function validateBatchParameters(input: DraftFormInput, errors: Record<string, string>): void {
  if (!input.targetTable || !input.targetTable.trim()) {
    errors.targetTable = 'Target table is required.';
  }

  if (!input.batchColumn || !input.batchColumn.trim()) {
    errors.batchColumn = 'Batch column is required.';
  }

  const parsedBatchSize = Number(input.batchSize);
  const isValidSize = Number.isFinite(parsedBatchSize) && parsedBatchSize >= 100 && parsedBatchSize <= 50000;

  if (!isValidSize) {
    errors.batchSize = 'Batch size must be between 100 and 50,000.';
  }
}
