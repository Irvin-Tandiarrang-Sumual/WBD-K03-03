import React from 'react';
import type { MigrationRecord } from '../../types';
import { formatMigrationVersion } from '../../utils/formatters';
import { M3Dialog } from '../common/M3Dialog';

interface SchemaDiffModalProps {
  migration: MigrationRecord | null;
  onClose: () => void;
}

export const SchemaDiffModal: React.FC<SchemaDiffModalProps> = ({ migration, onClose }) => {
  if (!migration) return null;

  return (
    <M3Dialog
      open={!!migration}
      onClose={onClose}
      title={`Schema Diff · ${formatMigrationVersion(migration.version)}`}
      subtitle={`Snapshot #${migration.diffFromVersion} → #${migration.version}`}
      maxWidth="max-w-2xl"
      footer={
        <button
          type="button"
          onClick={onClose}
          className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold shadow-sm transition-colors cursor-pointer"
        >
          Close
        </button>
      }
    >
      <div className="rounded-xl border-2 border-[#D8C7A5] bg-[#FFFFFF] overflow-hidden font-mono text-xs">
        {migration.schemaDiff.map((line, i) => (
          <div
            key={i}
            className={`px-4 py-2 border-b last:border-b-0 border-[#EFE5D1] whitespace-pre-wrap ${
              line.type === 'header'
                ? 'bg-[#EFE5D1]/70 text-[#1E6B8C] font-bold'
                : line.type === 'add'
                ? 'bg-[#264D3B]/12 text-[#1B3B2B] font-semibold'
                : line.type === 'remove'
                ? 'bg-[#822433]/12 text-[#822433] font-semibold'
                : 'text-[#6B5444]'
            }`}
          >
            {line.content}
          </div>
        ))}
      </div>
    </M3Dialog>
  );
};
