import React, { useEffect } from 'react';
import { X } from 'lucide-react';

interface M3DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  maxWidth?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}

export const M3Dialog: React.FC<M3DialogProps> = ({
  open,
  onClose,
  title,
  subtitle,
  maxWidth = 'max-w-xl',
  children,
  footer,
}) => {
  useEffect(() => {
    if (!open) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 bg-[#3A241A]/45 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={title}
      onClick={onClose}
    >
      <div
        className={`w-full ${maxWidth} bg-[#FFFDF9]/96 backdrop-blur-xl border-2 border-[#D8C7A5] border-t-4 border-t-[#264D3B] rounded-3xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 pt-5 pb-4 flex items-start justify-between gap-4 border-b border-[#EFE5D1] bg-[#F5EEDD]/45">
          <div className="min-w-0">
            <h2 className="text-[19px] font-semibold tracking-tight text-[#3A241A]">{title}</h2>
            {subtitle && <p className="text-xs font-mono text-[#6B5444] mt-1">{subtitle}</p>}
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={`Close ${title}`}
            className="w-10 h-10 -mr-1 rounded-full flex items-center justify-center text-[#6B5444] hover:text-[#3A241A] hover:bg-[#EFE5D1] transition-colors cursor-pointer shrink-0"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="px-6 py-5 overflow-y-auto flex-1">{children}</div>

        {footer && (
          <div className="px-6 py-4 border-t border-[#EFE5D1] bg-[#F5EEDD]/40 flex items-center justify-end gap-2.5">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
};
