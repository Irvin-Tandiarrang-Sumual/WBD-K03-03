import React from 'react';

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
}

export const Skeleton: React.FC<SkeletonProps> = ({ className = '', ...props }) => {
  return (
    <div
      aria-hidden="true"
      className={`animate-pulse bg-[#D8C7A5]/45 rounded-xl ${className}`}
      {...props}
    />
  );
};

export const PageHeaderSkeleton: React.FC = () => (
  <div className="flex flex-wrap items-center justify-between gap-4 pb-2">
    <div className="space-y-2">
      <Skeleton className="h-7 w-48 rounded-lg" />
      <Skeleton className="h-4 w-72 rounded-md" />
    </div>
    <div className="flex items-center gap-2">
      <Skeleton className="h-10 w-28 rounded-xl" />
      <Skeleton className="h-10 w-36 rounded-xl" />
    </div>
  </div>
);

export const TableSkeleton: React.FC<{ rows?: number; columns?: number }> = ({
  rows = 5,
  columns = 5,
}) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs">
    {/* Toolbar */}
    <div className="p-4 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3 rounded-t-2xl">
      <Skeleton className="h-10 w-64 rounded-xl" />
      <div className="flex items-center gap-2">
        <Skeleton className="h-10 w-28 rounded-xl" />
        <Skeleton className="h-10 w-28 rounded-xl" />
      </div>
    </div>

    {/* Table Header */}
    <div className="p-4 border-b border-[#EFE5D1] bg-[#F5EEDD]/40 flex items-center justify-between gap-4">
      {Array.from({ length: columns }).map((_, i) => (
        <Skeleton key={i} className="h-4 flex-1 rounded" />
      ))}
    </div>

    {/* Rows */}
    <div className="divide-y divide-[#EFE5D1] p-2">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="p-3 flex items-center justify-between gap-4">
          {Array.from({ length: columns }).map((_, c) => (
            <Skeleton key={c} className="h-5 flex-1 rounded-md" />
          ))}
        </div>
      ))}
    </div>
  </div>
);

export const CardGridSkeleton: React.FC<{ count?: number }> = ({ count = 6 }) => (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {Array.from({ length: count }).map((_, i) => (
      <div
        key={i}
        className="p-5 rounded-2xl bg-[#FFFDF9]/90 border border-[#D8C7A5] space-y-4"
      >
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-32 rounded-md" />
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
        <div className="space-y-2">
          <Skeleton className="h-3 w-44 rounded" />
          <Skeleton className="h-3 w-28 rounded" />
        </div>
        <div className="pt-3 border-t border-[#EFE5D1] flex items-center justify-between">
          <Skeleton className="h-4 w-24 rounded" />
          <Skeleton className="h-4 w-28 rounded" />
        </div>
      </div>
    ))}
  </div>
);
