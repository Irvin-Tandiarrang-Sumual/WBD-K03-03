import React, { useState } from 'react';
import { Check, Copy } from 'lucide-react';

interface SqlCodeViewerProps {
  code: string;
  title?: string;
  maxHeight?: string;
  className?: string;
  showCopy?: boolean;
}

export const SqlCodeViewer: React.FC<SqlCodeViewerProps> = ({
  code,
  title,
  maxHeight = 'max-h-[320px]',
  className = '',
  showCopy = true,
}) => {
  const [copied, setCopied] = useState(false);

  const lines = (code || '').split('\n');

  const handleCopy = () => {
    if (!code) return;
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={`rounded-xl border-2 border-[#D8C7A5] bg-[#FFFFFF] shadow-inner overflow-hidden ${className}`}
    >
      {title && (
        <div className="flex items-center justify-between px-3.5 py-2 bg-[#F5EEDD]/75 border-b border-[#D8C7A5] text-xs font-semibold text-[#3A241A]">
          <span className="font-mono text-[11px] tracking-wide">{title}</span>
          {showCopy && (
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium text-[#6B5444] hover:text-[#3A241A] hover:bg-[#EAE0CB] transition-colors"
              title="Copy SQL"
            >
              {copied ? (
                <>
                  <Check className="w-3 h-3 text-[#264D3B]" />
                  <span>Copied</span>
                </>
              ) : (
                <>
                  <Copy className="w-3 h-3" />
                  <span>Copy</span>
                </>
              )}
            </button>
          )}
        </div>
      )}

      <div className={`overflow-auto ${maxHeight} text-xs leading-relaxed font-mono p-3 flex`}>
        <div className="select-none text-right pr-3 pl-1 text-[#A89880] border-r border-[#E8DEC8] font-mono shrink-0">
          {lines.map((_, idx) => (
            <div key={idx} className="h-5 text-[11px] leading-5">
              {idx + 1}
            </div>
          ))}
        </div>

        <div className="pl-3 overflow-x-auto flex-1 text-[#3A241A] whitespace-pre">
          {lines.map((line, idx) => (
            <div key={idx} className="h-5 leading-5">
              {line || ' '}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
