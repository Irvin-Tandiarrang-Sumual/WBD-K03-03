import React, { useState } from 'react';
import type { DraftProposal } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

interface VetoReasonModalProps {
  draft: DraftProposal | null;
  onClose: () => void;
  onConfirmVeto: (draft: DraftProposal, reason: string) => void;
}

export const VetoReasonModal: React.FC<VetoReasonModalProps> = ({
  draft,
  onClose,
  onConfirmVeto,
}) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');

  if (!draft) return null;

  const handleConfirm = () => {
    if (!reason.trim()) {
      setError('Please provide a reason for vetoing this proposal.');
      return;
    }
    setError('');
    onConfirmVeto(draft, reason.trim());
  };

  return (
    <M3Dialog
      open={!!draft}
      onClose={onClose}
      title={`Reject / Veto ${draft.id}`}
      subtitle={draft.description}
      maxWidth="max-w-md"
      footer={
        <>
          <button
            type="button"
            onClick={onClose}
            className="h-10 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            className="h-10 px-5 rounded-xl bg-[#822433] hover:bg-[#691B28] text-white text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            Confirm Veto
          </button>
        </>
      }
    >
      <div className="space-y-3">
        <label htmlFor="veto-reason" className="block text-xs font-bold text-[#3A241A]">
          Rejection / Veto Reason
        </label>
        <textarea
          id="veto-reason"
          rows={4}
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            if (error) setError('');
          }}
          placeholder="Explain required schema changes, query optimization, or safety concerns..."
          className="w-full rounded-xl border-2 border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-sm p-3 focus:outline-none focus:border-[#822433] placeholder-[#8A7361]"
        />
        {error && <p className="text-xs font-semibold text-[#822433]">{error}</p>}
      </div>
    </M3Dialog>
  );
};
