import React from 'react';
import { Inbox } from 'lucide-react';

export interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  action,
  className = '',
}) => {
  return (
    <div
      className={`rounded-2xl border border-dashed border-[#D8C7A5] bg-[#FFFBF0]/60 p-12 text-center flex flex-col items-center justify-center space-y-3 ${className}`}
    >
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-[#F5EEDD] text-[#8C7664] shadow-inner">
        {icon || <Inbox className="h-6 w-6 text-[#8C7664]" />}
      </div>
      <div className="space-y-1">
        <h3 className="font-serif text-lg font-bold text-[#3A241A]">{title}</h3>
        {description && (
          <p className="text-sm text-[#6B5444] max-w-sm mx-auto">{description}</p>
        )}
      </div>
      {action && <div className="pt-2">{action}</div>}
    </div>
  );
};
