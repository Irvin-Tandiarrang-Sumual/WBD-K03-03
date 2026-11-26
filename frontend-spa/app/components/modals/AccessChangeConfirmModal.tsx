import React from 'react';
import { AlertTriangle, ShieldCheck } from 'lucide-react';
import type { DbPermission } from '../../types';
import { getPermissionChangeConfirmation } from '../../utils/permissions';
import { M3Dialog } from '../common/M3Dialog';

interface AccessChangeConfirmModalProps {
  changeData: {
    groupId: string;
    dbId: string;
    dbName: string;
    currentPermission: DbPermission;
    nextPermission: DbPermission;
  } | null;
  onClose: () => void;
  onConfirm: () => void;
}

export const AccessChangeConfirmModal: React.FC<AccessChangeConfirmModalProps> = ({
  changeData,
  onClose,
  onConfirm,
}) => {
  if (!changeData) return null;

  const copy = getPermissionChangeConfirmation(changeData.dbName, changeData.nextPermission);

  return (
    <M3Dialog
      open={!!changeData}
      onClose={onClose}
      title={copy.title}
      subtitle={`${changeData.dbName} · ${changeData.currentPermission} → ${changeData.nextPermission}`}
      maxWidth="max-w-lg"
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
            onClick={onConfirm}
            className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            <ShieldCheck className="w-4 h-4" />
            <span>Confirm Permission Update</span>
          </button>
        </>
      }
    >
      <div className="space-y-4">
        <p className="text-sm font-semibold text-[#3A241A]">{copy.question}</p>
        <div
          className={`rounded-xl p-4 border text-xs leading-relaxed flex items-start gap-3 ${
            changeData.nextPermission === 'view_edit'
              ? 'bg-[#F3B738]/20 border-[#F3B738]/70 text-[#3A241A]'
              : 'bg-[#822433]/10 border-[#822433]/35 text-[#822433]'
          }`}
        >
          <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
          <span className="font-medium">{copy.warning}</span>
        </div>
      </div>
    </M3Dialog>
  );
};
