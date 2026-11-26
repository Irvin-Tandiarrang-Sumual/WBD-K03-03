import React from 'react';
import { useLoaderData, useSearchParams } from 'react-router';
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Code2,
  Copy,
  Database,
  History,
  Play,
  RotateCcw,
  Sparkles,
  Table as TableIcon,
  Terminal,
} from 'lucide-react';
import { PageHeader } from '../components/common/PageHeader';
import { useQueryConsole, type QueryExecuteMode } from '../hooks/useQueryConsole';
import { databaseService } from '../services/database_service';
import type { TargetDatabase } from '../types/database';

export async function clientLoader() {
  const databases = await databaseService.getAccessibleDatabases();
  return { databases };
}

export default function QueryConsolePage() {
  const { databases } = useLoaderData() as { databases: TargetDatabase[] };
  const [searchParams] = useSearchParams();
  const dbParam = searchParams.get('db') || undefined;

  const {
    selectedDbId,
    setSelectedDbId,
    selectedDb,
    query,
    setQuery,
    isExecuting,
    executingMode,
    result,
    error,
    activeTab,
    setActiveTab,
    history,
    execute,
    handleClear,
    handleLoadSample,
    sampleQueries,
  } = useQueryConsole({ databases, initialDbId: dbParam });

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      execute('run');
    }
  };

  return (
    <section className="space-y-5">
      <PageHeader
        title="SQL Query Console"
        subtitle="Directly execute queries, generate execution plans, and inspect runtime database statistics"
        badge={{
          label: selectedDb ? `${selectedDb.name} (${selectedDb.engine})` : 'Console Ready',
          variant: 'success',
        }}
        actions={
          <div className="flex items-center gap-2">
            <select
              aria-label="Select Target Database"
              value={selectedDbId}
              onChange={(e) => setSelectedDbId(e.target.value)}
              className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
            >
              {databases.map((db) => (
                <option key={db.id} value={db.id}>
                  {db.name} ({db.engine} · {db.hostPort})
                </option>
              ))}
            </select>
          </div>
        }
      />

      {/* Editor & Action Bar Card */}
      <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
        {/* Sample Queries Bar */}
        <div className="px-4 py-2.5 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-bold text-[#6B5444] uppercase tracking-wider inline-flex items-center gap-1 mr-1">
              <Sparkles className="w-3.5 h-3.5 text-[#F3B738]" />
              Templates:
            </span>
            {sampleQueries.map((s, idx) => (
              <button
                key={idx}
                type="button"
                onClick={() => handleLoadSample(s.sql)}
                className="px-2.5 py-1 rounded-lg text-xs font-medium bg-[#FFFDF9] hover:bg-[#F5EEDD] border border-[#D8C7A5] text-[#3A241A] transition-colors cursor-pointer"
              >
                {s.label}
              </button>
            ))}
          </div>
          <div className="text-[11px] font-mono text-[#8A7361] hidden sm:block">
            Tip: Press <kbd className="px-1.5 py-0.5 rounded bg-[#EFE5D1] border border-[#D8C7A5] text-[#3A241A] font-bold">Ctrl + Enter</kbd> to execute
          </div>
        </div>

        {/* Text Area SQL Input */}
        <div className="p-4 relative">
          <textarea
            aria-label="SQL Query Input"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={5}
            placeholder="Type your SQL query here (e.g. SELECT * FROM users LIMIT 10;)"
            className="w-full font-mono text-sm leading-relaxed text-[#3A241A] bg-[#FFFBF0]/60 p-3.5 rounded-xl border border-[#D8C7A5] focus:outline-none focus:border-[#264D3B] focus:ring-1 focus:ring-[#264D3B] resize-y"
          />
        </div>

        {/* Control Buttons Footer */}
        <div className="px-4 py-3 bg-[#F5EEDD]/45 border-t border-[#EFE5D1] flex flex-wrap items-center justify-between gap-3">
          <button
            type="button"
            onClick={handleClear}
            className="min-h-[44px] h-11 px-4 rounded-xl border border-[#D8C7A5] hover:bg-[#EFE5D1] text-[#6B5444] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer focus-visible:ring-2 focus-visible:ring-[#264D3B]"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            <span>Clear</span>
          </button>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={isExecuting || !query.trim()}
              onClick={() => execute('explain')}
              className="min-h-[44px] h-11 px-4 rounded-xl border border-[#1E6B8C]/40 bg-[#1E6B8C]/10 hover:bg-[#1E6B8C]/20 text-[#1E6B8C] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-[#1E6B8C]"
            >
              <Code2 className="w-3.5 h-3.5" />
              <span>{executingMode === 'explain' ? 'Explaining...' : 'Explain Plan'}</span>
            </button>

            <button
              type="button"
              disabled={isExecuting || !query.trim()}
              onClick={() => execute('explain-analyze')}
              className="min-h-[44px] h-11 px-4 rounded-xl border border-[#F3B738]/60 bg-[#F3B738]/15 hover:bg-[#F3B738]/25 text-[#6E4900] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-[#F3B738]"
            >
              <Sparkles className="w-3.5 h-3.5 text-[#8A5A00]" />
              <span>{executingMode === 'explain-analyze' ? 'Analyzing...' : 'Explain Analyze'}</span>
            </button>

            <button
              type="button"
              disabled={isExecuting || !query.trim()}
              onClick={() => execute('run')}
              className="min-h-[44px] h-11 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-[#264D3B]"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>{executingMode === 'run' ? 'Running...' : 'Run Query'}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="rounded-2xl bg-[#822433]/10 border border-[#822433]/30 p-4 text-[#822433] flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5 text-[#822433]" />
          <div className="min-w-0">
            <h4 className="text-xs font-bold uppercase tracking-wider">Query Execution Failed</h4>
            <pre className="font-mono text-xs whitespace-pre-wrap mt-1 leading-relaxed">{error}</pre>
          </div>
        </div>
      )}

      {/* Query Result Card */}
      {result && (
        <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
          {/* Result Header & Stats */}
          <div className="px-4 py-3 bg-[#EFE5D1]/60 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <span className="inline-flex items-center gap-1 text-xs font-bold text-[#264D3B]">
                <CheckCircle2 className="w-4 h-4 text-[#264D3B]" />
                Success
              </span>
              <span className="inline-flex items-center gap-1 text-xs font-mono text-[#6B5444]">
                <Clock className="w-3.5 h-3.5" />
                {result.latencyMs ?? 0}ms
              </span>
              <span className="text-xs font-mono text-[#6B5444]">
                {result.rows.length} {result.rows.length === 1 ? 'row' : 'rows'}
              </span>
              <span className="text-xs font-mono text-[#6B5444]">
                {result.columns.length} columns
              </span>
            </div>

            <div className="inline-flex rounded-lg border border-[#D8C7A5] bg-[#FFFDF9] p-0.5">
              <button
                type="button"
                onClick={() => setActiveTab('table')}
                className={`h-7 px-2.5 rounded-md text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer ${
                  activeTab === 'table' ? 'bg-[#264D3B] text-[#F5EEDD]' : 'text-[#6B5444] hover:bg-[#EFE5D1]'
                }`}
              >
                <TableIcon className="w-3.5 h-3.5" />
                <span>Table</span>
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('json')}
                className={`h-7 px-2.5 rounded-md text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer ${
                  activeTab === 'json' ? 'bg-[#264D3B] text-[#F5EEDD]' : 'text-[#6B5444] hover:bg-[#EFE5D1]'
                }`}
              >
                <Code2 className="w-3.5 h-3.5" />
                <span>JSON</span>
              </button>
            </div>
          </div>

          {/* Result Body */}
          {activeTab === 'table' ? (
            <div className="overflow-x-auto max-h-[500px]">
              {result.rows.length === 0 ? (
                <div className="py-12 text-center text-xs font-mono text-[#8A7361]">
                  Query executed successfully. 0 rows returned.
                </div>
              ) : (
                <table className="w-full text-left border-collapse font-mono text-xs">
                  <thead className="sticky top-0 bg-[#F5EEDD]/95 border-b border-[#D8C7A5] z-10 backdrop-blur-xs">
                    <tr>
                      <th className="py-2.5 px-3 text-[10px] font-bold text-[#8A7361] border-r border-[#EFE5D1] w-12 text-center">
                        #
                      </th>
                      {result.columns.map((col, idx) => (
                        <th key={idx} className="py-2.5 px-3 font-bold text-[#3A241A] border-r border-[#EFE5D1] last:border-r-0">
                          {col}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#EFE5D1]">
                    {result.rows.map((row, rowIdx) => (
                      <tr key={rowIdx} className="hover:bg-[#F5EEDD]/45 transition-colors">
                        <td className="py-2 px-3 text-[#8A7361] text-center border-r border-[#EFE5D1]">
                          {rowIdx + 1}
                        </td>
                        {result.columns.map((col, colIdx) => {
                          const val = row[col];
                          const displayVal = val === null || val === undefined
                            ? '<null>'
                            : typeof val === 'object'
                            ? JSON.stringify(val)
                            : String(val);
                          return (
                            <td
                              key={colIdx}
                              className={`py-2 px-3 border-r border-[#EFE5D1] last:border-r-0 truncate max-w-xs ${
                                val === null || val === undefined ? 'text-[#8A7361] italic' : 'text-[#3A241A]'
                              }`}
                              title={displayVal}
                            >
                              {displayVal}
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          ) : (
            <div className="p-4 bg-[#FFFBF0]/60 overflow-x-auto max-h-[500px]">
              <pre className="font-mono text-xs text-[#3A241A] leading-relaxed">
                {JSON.stringify(result.rows, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* Query History Panel */}
      {history.length > 0 && (
        <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] p-4 shadow-xs">
          <div className="flex items-center gap-1.5 mb-2.5">
            <History className="w-4 h-4 text-[#6B5444]" />
            <h3 className="text-xs font-bold uppercase tracking-wider text-[#3A241A]">Recent Queries in this Session</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {history.map((hist, idx) => (
              <button
                key={idx}
                type="button"
                onClick={() => setQuery(hist)}
                className="px-3 py-1.5 rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/60 hover:bg-[#EFE5D1] font-mono text-xs text-[#3A241A] text-left max-w-md truncate transition-colors cursor-pointer"
                title={hist}
              >
                {hist}
              </button>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
