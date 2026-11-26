import React from 'react';
import {
  AlertTriangle,
  Check,
  CheckCircle2,
  Clock,
  Lock,
  RefreshCw,
  XCircle,
} from 'lucide-react';
import type { DraftStatus } from '../../types';

interface DraftStatusChipProps {
  status: DraftStatus;
  className?: string;
}

export const DraftStatusChip: React.FC<DraftStatusChipProps> = ({ status, className = '' }) => {
  const map: Record<DraftStatus, { label: string; cls: string; icon: React.ReactNode }> = {
    pending_approval: {
      label: 'pending_approval',
      cls: 'bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70',
      icon: <Clock className="w-3.5 h-3.5 text-[#8A5A00]" />,
    },
    approved: {
      label: 'approved',
      cls: 'bg-[#264D3B]/15 text-[#264D3B] border border-[#264D3B]/35',
      icon: <CheckCircle2 className="w-3.5 h-3.5" />,
    },
    applied: {
      label: 'applied',
      cls: 'bg-[#264D3B] text-[#F5EEDD] border border-[#1B3B2B]',
      icon: <Check className="w-3.5 h-3.5" />,
    },
    executing: {
      label: 'executing',
      cls: 'bg-[#1E6B8C]/15 text-[#1E6B8C] border border-[#1E6B8C]/35',
      icon: <RefreshCw className="w-3.5 h-3.5 animate-spin" />,
    },
    partial: {
      label: 'partial (recovery)',
      cls: 'bg-[#F3B738]/30 text-[#6E4900] border border-[#F3B738]',
      icon: <AlertTriangle className="w-3.5 h-3.5 text-[#8A5A00]" />,
    },
    partially_reverted: {
      label: 'partially_reverted',
      cls: 'bg-[#822433] text-white border border-[#691B28]',
      icon: <AlertTriangle className="w-3.5 h-3.5" />,
    },
    rejected: {
      label: 'rejected',
      cls: 'bg-[#822433]/15 text-[#822433] border border-[#822433]/35',
      icon: <XCircle className="w-3.5 h-3.5" />,
    },
    failed: {
      label: 'failed',
      cls: 'bg-[#822433] text-white border border-[#691B28]',
      icon: <XCircle className="w-3.5 h-3.5" />,
    },
    invalidated: {
      label: 'invalidated',
      cls: 'bg-[#EFE5D1] text-[#6B5444] border border-[#D8C7A5]',
      icon: <Lock className="w-3.5 h-3.5" />,
    },
  };

  const cfg = map[status] || {
    label: status,
    cls: 'bg-[#EFE5D1] text-[#6B5444] border border-[#D8C7A5]',
    icon: <Lock className="w-3.5 h-3.5" />,
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold whitespace-nowrap ${cfg.cls} ${className}`}
    >
      {cfg.icon}
      {cfg.label}
    </span>
  );
};
