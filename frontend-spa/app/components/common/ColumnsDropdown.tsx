import React, { useEffect, useRef, useState } from 'react';
import { Check, ChevronDown, Columns3 } from 'lucide-react';

export interface ColumnOption {
  key: string;
  label: string;
  visible?: boolean;
}

export interface ColumnsDropdownProps {
  columns: ColumnOption[];
  visibleMap?: Record<string, boolean>;
  onToggle: (key: string) => void;
}

export const ColumnsDropdown: React.FC<ColumnsDropdownProps> = ({
  columns,
  visibleMap,
  onToggle,
}) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', handleOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('mousedown', handleOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  }, [open]);

  const isChecked = (col: ColumnOption) =>
    visibleMap ? !!visibleMap[col.key] : col.visible !== false;

  const activeCount = columns.filter(isChecked).length;

  return (
    <div ref={ref} className="relative inline-block text-left">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label="Toggle visible columns"
        className="h-10 px-3.5 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-2 transition-colors cursor-pointer"
      >
        <Columns3 className="w-4 h-4 text-[#1E6B8C]" />
        <span>Columns</span>
        <span className="font-mono text-[11px] text-[#6B5444]">
          {activeCount}/{columns.length}
        </span>
        <ChevronDown className="w-3.5 h-3.5 text-[#6B5444]" />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 mt-2 w-52 rounded-2xl bg-[#FFFDF9]/98 backdrop-blur-md border border-[#D8C7A5] shadow-xl py-2 z-40"
        >
          <div className="px-3.5 py-1.5 text-[10px] font-bold uppercase tracking-wider text-[#6B5444] border-b border-[#EFE5D1]">
            Visible Columns
          </div>
          <div className="py-1">
            {columns.map((col) => {
              const checked = isChecked(col);
              const isLast = checked && activeCount <= 1;
              return (
                <button
                  key={col.key}
                  type="button"
                  role="menuitemcheckbox"
                  aria-checked={checked}
                  disabled={isLast}
                  onClick={() => onToggle(col.key)}
                  className={`w-full px-3.5 py-2 text-left text-xs flex items-center justify-between gap-2 transition-colors ${
                    isLast
                      ? 'opacity-50 cursor-not-allowed'
                      : 'hover:bg-[#F5EEDD] cursor-pointer text-[#3A241A]'
                  }`}
                >
                  <span className="font-medium">{col.label}</span>
                  <span
                    className={`w-4 h-4 rounded flex items-center justify-center border transition-colors ${
                      checked
                        ? 'bg-[#264D3B] border-[#264D3B] text-[#F5EEDD]'
                        : 'border-[#D8C7A5] bg-white'
                    }`}
                  >
                    {checked && <Check className="w-3 h-3 stroke-[2.8]" />}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
