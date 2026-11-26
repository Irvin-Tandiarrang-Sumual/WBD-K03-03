import React from 'react';
import { FileDiff, XCircle } from 'lucide-react';
import type { MigrationRecord } from '../../types';
import { formatMigrationVersion } from '../../utils/formatters';
import { M3Dialog } from '../common/M3Dialog';
import { SqlCodeViewer } from '../code/SqlCodeViewer';

interface MigrationDetailModalProps {
  migration: MigrationRecord | null;
  onClose: () => void;
  onOpenDiff: (migration: MigrationRecord) => void;
}

export const MigrationDetailModal: React.FC<MigrationDetailModalProps> = ({
  migration,
  onClose,
  onOpenDiff,
}) => {
  if (!migration) return null;

  return (
    <M3Dialog
      open={!!migration}
      onClose={onClose}
      title={formatMigrationVersion(migration.version)}
      subtitle={`${migration.type} · ${migration.status} · ${migration.executedBy} · ${migration.executedAtRelative}`}
      maxWidth="max-w-2xl"
      footer={
        <>
          <button
            type="button"
            onClick={() => onOpenDiff(migration)}
            className="h-10 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <FileDiff className="w-4 h-4 text-[#264D3B]" />
            <span>Schema Diff</span>
          </button>
          <button
            type="button"
            onClick={onClose}
            className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold shadow-sm transition-colors cursor-pointer"
          >
            Close
          </button>
        </>
      }
    >
      <div className="space-y-4">
        {migration.status === 'Fail' && migration.errorLog && (
          <div className="rounded-2xl bg-[#822433] text-white p-4 shadow-sm">
            <div className="flex items-center gap-2 text-xs font-bold text-[#F3B738] mb-1.5">
              <XCircle className="w-4 h-4" />
              <span>Driver Error</span>
            </div>
            <pre className="font-mono text-xs whitespace-pre-wrap leading-relaxed text-[#FFFDF9]">
              {migration.errorLog}
            </pre>
          </div>
        )}

        <div>
          <div className="text-xs font-bold text-[#3A241A] mb-2">Up Script (SQL)</div>
          <SqlCodeViewer code={migration.upSql} maxHeight="max-h-[220px]" />
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-bold text-[#3A241A]">Down Script (SQL)</span>
            <span className="font-mono text-[11px] text-[#6B5444]">System-handled</span>
          </div>
          {migration.downSql ? (
            <SqlCodeViewer code={migration.downSql} maxHeight="max-h-[220px]" />
          ) : (
            <div className="rounded-xl border border-[#F3B738]/60 bg-[#F3B738]/15 px-4 py-3 text-xs font-semibold text-[#6E4900]">
              Irreversible migration · No automatic rollback script generated.
            </div>
          )}
        </div>
      </div>
    </M3Dialog>
  );
};
