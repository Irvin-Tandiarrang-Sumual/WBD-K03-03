import React from 'react';

export interface PageHeaderProps {
  title: string;
  subtitle?: string;
  description?: string;
  badge?:
    | {
        label: string;
        variant?: 'success' | 'warning' | 'danger' | 'neutral';
      }
    | React.ReactNode;
  action?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

const getBadgeStyle = (variant: 'success' | 'warning' | 'danger' | 'neutral' = 'neutral') => {
  switch (variant) {
    case 'success':
      return 'bg-[#264D3B]/15 text-[#264D3B] border border-[#264D3B]/30';
    case 'warning':
      return 'bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70';
    case 'danger':
      return 'bg-[#822433]/15 text-[#822433] border border-[#822433]/30';
    case 'neutral':
    default:
      return 'bg-[#EFE5D1] text-[#3A241A] border border-[#D8C7A5]';
  }
};

export const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  subtitle,
  description,
  badge,
  action,
  actions,
  className = '',
}) => {
  const sub = subtitle || description;
  const rightActions = actions || action;

  return (
    <div className={`flex flex-wrap items-center justify-between gap-3 ${className}`}>
      <div className="space-y-0.5 min-w-0">
        <div className="flex items-center gap-2.5 flex-wrap">
          <h1 className="text-2xl font-bold tracking-tight text-[#3A241A] truncate">{title}</h1>
          {badge && (
            <div>
              {typeof badge === 'object' && 'label' in badge ? (
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold ${getBadgeStyle(
                    badge.variant
                  )}`}
                >
                  {badge.label}
                </span>
              ) : (
                badge
              )}
            </div>
          )}
        </div>
        {sub && <p className="text-xs text-[#6B5444] font-medium">{sub}</p>}
      </div>
      {rightActions && <div className="shrink-0 flex items-center gap-2">{rightActions}</div>}
    </div>
  );
};
