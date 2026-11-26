import React from 'react';
import { useLoaderData, useParams, Link } from 'react-router';
import { Check, Send } from 'lucide-react';
import { PageHeader } from '../components/common/PageHeader';
import { useCreateDraft } from '../hooks/useCreateDraft';
import { databaseService } from '../services/database_service';
import type { DraftFormInput, DraftProposalMode, TargetDatabase } from '../types';

export async function clientLoader() {
  const databases = await databaseService.getAccessibleDatabases();
  return { databases };
}

export default function CreateDraftPage() {
  const { databases } = useLoaderData() as { databases: TargetDatabase[] };
  const params = useParams();

  const defaultDbId = params.id || databases[0]?.id || 'db-inventory-pg';
  const d = useCreateDraft(defaultDbId);

  return (
    <section className="space-y-5 max-w-4xl mx-auto">
      <PageHeader
        title="Create Migration Draft"
        subtitle={`Target Database: ${d.form.databaseId}`}
        badge={{ label: 'Consensus Required', variant: 'warning' }}
      />

      <DraftProposalForm
        form={d.form}
        databases={databases}
        errors={d.errors}
        isSubmitting={d.isSubmitting}
        onUpdateField={d.updateField}
        onSubmit={d.handleSubmit}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface DraftProposalFormProps {
  form: DraftFormInput;
  databases: TargetDatabase[];
  errors: Record<string, string>;
  isSubmitting: boolean;
  onUpdateField: <K extends keyof DraftFormInput>(key: K, value: DraftFormInput[K]) => void;
  onSubmit: (e: React.FormEvent) => void;
}

const DraftProposalForm: React.FC<DraftProposalFormProps> = ({
  form,
  databases,
  errors,
  isSubmitting,
  onUpdateField,
  onSubmit,
}) => (
  <form
    onSubmit={onSubmit}
    className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs p-6 space-y-5"
  >
    <div>
      <label htmlFor="draft-database" className="block text-xs font-bold text-[#3A241A] mb-1.5">
        Target Database
      </label>
      <select
        id="draft-database"
        value={form.databaseId}
        onChange={(e) => onUpdateField('databaseId', e.target.value)}
        className="w-full h-11 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3.5 focus:outline-none focus:border-[#1E6B8C]"
      >
        {databases.map((db) => (
          <option key={db.id} value={db.id}>
            {db.name} ({db.engine} · {db.hostPort}) {db.quarantined ? ' [423 Quarantined]' : ''}
          </option>
        ))}
      </select>
    </div>

    <div>
      <label htmlFor="draft-description" className="block text-xs font-bold text-[#3A241A] mb-1.5">
        Proposal Summary / Title
      </label>
      <input
        id="draft-description"
        type="text"
        value={form.description}
        onChange={(e) => onUpdateField('description', e.target.value)}
        placeholder="Add tracking columns and backfill historical audit records"
        className="w-full h-11 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
      />
      {errors.description && (
        <p className="text-xs font-semibold text-[#822433] mt-1">{errors.description}</p>
      )}
    </div>

    <ModeToggle
      mode={form.mode}
      onSelectMode={(mode) => onUpdateField('mode', mode)}
    />

    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label htmlFor="draft-up-sql" className="text-xs font-bold text-[#3A241A]">
          Up Migration SQL Script
        </label>
        <span className="text-[11px] font-mono text-[#6B5444]">Rollback auto-managed</span>
      </div>
      <textarea
        id="draft-up-sql"
        rows={7}
        value={form.upSql}
        onChange={(e) => onUpdateField('upSql', e.target.value)}
        placeholder={`ALTER TABLE orders ADD COLUMN tracking_code VARCHAR(64) NULL;\nCREATE INDEX idx_orders_tracking ON orders (tracking_code);`}
        className="w-full bg-[#FFFFFF] border-2 border-[#D8C7A5] text-[#3A241A] font-mono text-xs leading-relaxed shadow-inner rounded-xl p-4 focus:outline-none focus:border-[#1E6B8C]"
      />
      {errors.upSql && (
        <p className="text-xs font-semibold text-[#822433] mt-1">{errors.upSql}</p>
      )}
    </div>

    {form.mode === 'batch_dml' && (
      <BatchParametersFieldset
        targetTable={form.targetTable}
        onTargetTableChange={(val) => onUpdateField('targetTable', val)}
        batchColumn={form.batchColumn}
        onBatchColumnChange={(val) => onUpdateField('batchColumn', val)}
        batchSize={form.batchSize}
        onBatchSizeChange={(val) => onUpdateField('batchSize', val)}
        errors={errors}
      />
    )}

    <div className="pt-3 border-t border-[#EFE5D1] flex items-center justify-end gap-3">
      <Link
        to="/drafts"
        className="min-h-[44px] h-11 px-5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold inline-flex items-center cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B]"
      >
        Cancel
      </Link>
      <button
        type="submit"
        disabled={isSubmitting}
        className="min-h-[44px] h-11 px-6 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] disabled:opacity-50 text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-2 shadow-sm transition-colors cursor-pointer focus-visible:ring-2 focus-visible:ring-[#264D3B]"
      >
        <Send className="w-4 h-4" />
        <span>{isSubmitting ? 'Submitting Proposal...' : 'Submit Draft for Quorum'}</span>
      </button>
    </div>
  </form>
);

interface ModeToggleProps {
  mode: DraftProposalMode;
  onSelectMode: (mode: DraftProposalMode) => void;
}

const MODE_OPTIONS = [
  { id: 'normal' as const, label: 'Normal DDL Migration' },
  { id: 'batch_dml' as const, label: 'Batch DML Migration' },
];

const ModeToggle: React.FC<ModeToggleProps> = ({ mode, onSelectMode }) => (
  <div>
    <div className="text-xs font-bold text-[#3A241A] mb-1.5">Migration Execution Mode</div>
    <div className="inline-flex rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/70 p-1">
      {MODE_OPTIONS.map((opt) => {
        const active = mode === opt.id;
        const btnClass = active
          ? 'bg-[#1E6B8C] text-white shadow-xs'
          : 'text-[#3A241A] hover:bg-[#EFE5D1]';

        return (
          <button
            key={opt.id}
            type="button"
            onClick={() => onSelectMode(opt.id)}
            className={`h-9 px-4 rounded-lg text-xs font-semibold inline-flex items-center gap-1.5 transition-all cursor-pointer ${btnClass}`}
          >
            {active && <Check className="w-3.5 h-3.5" />}
            <span>{opt.label}</span>
          </button>
        );
      })}
    </div>
  </div>
);

interface BatchParametersFieldsetProps {
  targetTable: string;
  onTargetTableChange: (value: string) => void;
  batchColumn: string;
  onBatchColumnChange: (value: string) => void;
  batchSize: string;
  onBatchSizeChange: (value: string) => void;
  errors: Record<string, string>;
}

const BatchParametersFieldset: React.FC<BatchParametersFieldsetProps> = ({
  targetTable,
  onTargetTableChange,
  batchColumn,
  onBatchColumnChange,
  batchSize,
  onBatchSizeChange,
  errors,
}) => (
  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 p-4 rounded-xl bg-[#F5EEDD]/65 border border-[#D8C7A5]">
    <div>
      <label htmlFor="batch-target-table" className="block text-xs font-bold text-[#3A241A] mb-1">
        Target Table
      </label>
      <input
        id="batch-target-table"
        type="text"
        value={targetTable}
        onChange={(e) => onTargetTableChange(e.target.value)}
        className="w-full h-10 font-mono text-xs rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] px-3 focus:outline-none focus:border-[#1E6B8C]"
      />
      {errors.targetTable && (
        <p className="text-[11px] font-semibold text-[#822433] mt-1">{errors.targetTable}</p>
      )}
    </div>
    <div>
      <label htmlFor="batch-column" className="block text-xs font-bold text-[#3A241A] mb-1">
        Batch Column Key
      </label>
      <input
        id="batch-column"
        type="text"
        value={batchColumn}
        onChange={(e) => onBatchColumnChange(e.target.value)}
        className="w-full h-10 font-mono text-xs rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] px-3 focus:outline-none focus:border-[#1E6B8C]"
      />
      {errors.batchColumn && (
        <p className="text-[11px] font-semibold text-[#822433] mt-1">{errors.batchColumn}</p>
      )}
    </div>
    <div>
      <label htmlFor="batch-size" className="block text-xs font-bold text-[#3A241A] mb-1">
        Chunk Size (Rows)
      </label>
      <input
        id="batch-size"
        type="number"
        value={batchSize}
        onChange={(e) => onBatchSizeChange(e.target.value)}
        className="w-full h-10 font-mono text-xs rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] px-3 focus:outline-none focus:border-[#1E6B8C]"
      />
      {errors.batchSize && (
        <p className="text-[11px] font-semibold text-[#822433] mt-1">{errors.batchSize}</p>
      )}
    </div>
  </div>
);
