import React from 'react';
import { ArrowRight, ShieldAlert } from 'lucide-react';
import { Link } from 'react-router';

interface QuarantineBannerProps {
  databaseName?: string;
  quarantineDraftId?: string;
  className?: string;
}

export const QuarantineBanner: React.FC<QuarantineBannerProps> = ({
  databaseName,
  quarantineDraftId,
  className = '',
}) => {
  return (
    <div
      role="alert"
      className={`bg-[#822433] text-white shadow-md rounded-xl p-4 flex flex-wrap items-center justify-between gap-3 ${className}`}
    >
      <div className="flex items-center gap-3">
        <ShieldAlert className="w-5 h-5 shrink-0 text-[#F5EEDD]" />
        <div className="flex flex-wrap items-center gap-2 text-xs sm:text-sm">
          <span className="px-2 py-0.5 rounded-md bg-white/15 font-mono text-[11px] font-bold tracking-wide">
            423 LOCKED
          </span>
          <span className="font-medium">
            Database {databaseName ? `"${databaseName}" ` : ''}Quarantined: Schema execution locked
            until partial rollback or draft failure is resolved.
          </span>
        </div>
      </div>
      {quarantineDraftId && (
        <Link
          to={`/drafts/${quarantineDraftId}`}
          className="h-9 px-4 rounded-full bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#822433] text-xs font-bold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer shrink-0"
        >
          <span>Resolve {quarantineDraftId}</span>
          <ArrowRight className="w-3.5 h-3.5" />
        </Link>
      )}
    </div>
  );
};
