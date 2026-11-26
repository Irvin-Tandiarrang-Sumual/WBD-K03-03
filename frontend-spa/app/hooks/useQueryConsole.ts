import { useState, useMemo, useCallback } from 'react';
import { databaseService } from '../services/database_service';
import type { DatabaseQueryResult, TargetDatabase } from '../types/database';

export type QueryExecuteMode = 'run' | 'explain' | 'explain-analyze';

interface UseQueryConsoleOptions {
  databases: TargetDatabase[];
  initialDbId?: string;
}

const DEFAULT_SAMPLE_QUERIES = [
  { label: 'Version & Health', sql: 'SELECT 1 AS status, CURRENT_TIMESTAMP AS server_time;' },
  { label: 'Information Schema Tables', sql: 'SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN (\'pg_catalog\', \'information_schema\') LIMIT 15;' },
  { label: 'Current User & Database', sql: 'SELECT current_user, current_database();' },
];

export function useQueryConsole({ databases, initialDbId }: UseQueryConsoleOptions) {
  const [selectedDbId, setSelectedDbId] = useState<string>(() => {
    if (initialDbId && databases.some((d) => d.id === initialDbId)) {
      return initialDbId;
    }
    return databases[0]?.id || '';
  });

  const [query, setQuery] = useState<string>('SELECT 1 AS status, CURRENT_TIMESTAMP AS ping;');
  const [isExecuting, setIsExecuting] = useState<boolean>(false);
  const [executingMode, setExecutingMode] = useState<QueryExecuteMode | null>(null);
  const [result, setResult] = useState<DatabaseQueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'table' | 'json'>('table');
  const [history, setHistory] = useState<string[]>([]);

  const selectedDb = useMemo(() => {
    return databases.find((d) => d.id === selectedDbId) || databases[0];
  }, [databases, selectedDbId]);

  const execute = useCallback(
    async (mode: QueryExecuteMode = 'run') => {
      const trimmed = query.trim();
      if (!trimmed || !selectedDbId) return;

      setIsExecuting(true);
      setExecutingMode(mode);
      setError(null);

      try {
        let res: DatabaseQueryResult;
        if (mode === 'explain') {
          res = await databaseService.explainQuery(selectedDbId, trimmed);
        } else if (mode === 'explain-analyze') {
          res = await databaseService.explainAnalyzeQuery(selectedDbId, trimmed);
        } else {
          res = await databaseService.executeQuery(selectedDbId, trimmed);
        }
        setResult(res);
        setHistory((prev) => {
          const filtered = prev.filter((q) => q !== trimmed);
          return [trimmed, ...filtered].slice(0, 10);
        });
      } catch (err: any) {
        setError(err.message || 'Failed to execute query');
        setResult(null);
      } finally {
        setIsExecuting(false);
        setExecutingMode(null);
      }
    },
    [query, selectedDbId]
  );

  const handleClear = () => {
    setQuery('');
    setResult(null);
    setError(null);
  };

  const handleLoadSample = (sampleSql: string) => {
    setQuery(sampleSql);
  };

  return {
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
    sampleQueries: DEFAULT_SAMPLE_QUERIES,
  };
}
