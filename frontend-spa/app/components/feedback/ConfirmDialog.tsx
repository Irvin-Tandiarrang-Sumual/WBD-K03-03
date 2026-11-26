import React from 'react';
import { AlertTriangle } from 'lucide-react';
import { M3Dialog } from '../common/M3Dialog';

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  isDestructive?: boolean;
  isLoading?: boolean;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  isDestructive = false,
  isLoading = false,
}) => {
  return (
    <M3Dialog
      open={open}
      onClose={onClose}
      title={title}
      maxWidth="max-w-md"
      footer={
        <>
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="min-h-[44px] h-11 px-5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold cursor-pointer disabled:opacity-50 transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B]"
          >
            {cancelText}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isLoading}
            className={`min-h-[44px] h-11 px-5 rounded-xl text-xs font-semibold inline-flex items-center gap-2 cursor-pointer shadow-sm transition-colors disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-[#264D3B] ${
              isDestructive
                ? 'bg-[#822433] hover:bg-[#681C28] text-white'
                : 'bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD]'
            }`}
          >
            {isLoading && (
              <span className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin shrink-0" />
            )}
            <span>{confirmText}</span>
          </button>
        </>
      }
    >
      <div className="flex items-start gap-3.5 py-1">
        {isDestructive && (
          <div className="w-10 h-10 rounded-full bg-[#822433]/15 text-[#822433] flex items-center justify-center shrink-0">
            <AlertTriangle className="w-5 h-5" />
          </div>
        )}
        <div className="flex-1">
          <p className="text-sm text-[#3A241A] leading-relaxed whitespace-pre-wrap">{message}</p>
        </div>
      </div>
    </M3Dialog>
  );
};
