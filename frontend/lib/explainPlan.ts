import type { QueryResponse } from "@/lib/types";

export interface ExecutionPlanAnalysis {
  isAnalyze: boolean;
  rawText: string;
  scanType?: string;
  indexUsed?: string;
  joinStrategy?: string;
  estimatedCost?: string;
  actualTime?: string;
  rowsProcessed?: string;
}

export function parseExecutionPlan(
  response: QueryResponse,
  analyze: boolean,
  dbType?: string,
): ExecutionPlanAnalysis {
  const rows = response.data?.rows || [];
  const columns =
    response.data?.columns && response.data.columns.length > 0
      ? response.data.columns
      : rows.length > 0
        ? Object.keys(rows[0])
        : [];

  const textLines: string[] = [];
  let mysqlScanType: string | undefined;
  let mysqlIndexUsed: string | undefined;
  let mysqlRows: string | undefined;

  if (columns.length > 1) {
    // Tabular EXPLAIN output (e.g. standard MySQL EXPLAIN)
    for (const row of rows) {
      const rowType = row.type ?? row.TYPE;
      const rowKey = row.key ?? row.KEY;
      const rowRows = row.rows ?? row.ROWS;
      const rowTable = row.table ?? row.TABLE;
      const rowExtra = row.Extra ?? row.extra ?? row.EXTRA;

      if (rowType && !mysqlScanType) {
        const t = String(rowType);
        mysqlScanType =
          t === "ALL"
            ? "ALL (Full Table Scan)"
            : t === "index"
              ? "Full Index Scan"
              : t === "range"
                ? "Index Range Scan"
                : t === "ref" || t === "eq_ref"
                  ? `Index Lookup (${t})`
                  : t === "const" || t === "system"
                    ? `Constant Lookup (${t})`
                    : t;
      }
      if (rowKey && !mysqlIndexUsed) {
        mysqlIndexUsed = String(rowKey);
      }
      if (rowRows !== undefined && rowRows !== null && !mysqlRows) {
        mysqlRows = String(rowRows);
      }

      const summaryParts = columns
        .filter((col) => row[col] !== undefined && row[col] !== null && String(row[col]) !== "")
        .map((col) => `${col}: ${String(row[col])}`);
      if (summaryParts.length > 0) {
        textLines.push(
          rowTable
            ? `[Table ${String(rowTable)}] ${summaryParts.join(" | ")}${rowExtra ? ` (${String(rowExtra)})` : ""}`
            : summaryParts.join(" | "),
        );
      }
    }
  } else {
    // Single-column text plan (PostgreSQL EXPLAIN / EXPLAIN ANALYZE, MySQL EXPLAIN ANALYZE)
    for (const row of rows) {
      for (const col of columns) {
        if (row[col] !== undefined && row[col] !== null) {
          textLines.push(String(row[col]));
        }
      }
    }
  }

  const fullText = textLines.join("\n");

  let scanType = mysqlScanType;
  if (!scanType) {
    if (/Index Only Scan/i.test(fullText)) scanType = "Index Only Scan";
    else if (/Index Scan/i.test(fullText)) scanType = "Index Scan";
    else if (/Bitmap Heap Scan/i.test(fullText)) scanType = "Bitmap Heap Scan";
    else if (/Seq Scan|Table scan/i.test(fullText))
      scanType = "Sequential Scan (Table Scan)";
    else if (/ALL/i.test(fullText) && /mysql/i.test(dbType || ""))
      scanType = "ALL (Full Table Scan)";
    else if (/Index lookup|ref|eq_ref/i.test(fullText))
      scanType = "Index Lookup (ref)";
  }

  let indexUsed = mysqlIndexUsed;
  if (!indexUsed) {
    const idxMatch = fullText.match(
      /using\s+([a-zA-Z0-9_]+)|key:\s*([a-zA-Z0-9_]+)/i,
    );
    if (idxMatch) {
      indexUsed = idxMatch[1] || idxMatch[2];
    }
  }

  let joinStrategy: string | undefined;
  if (/Hash Join/i.test(fullText)) joinStrategy = "Hash Join";
  else if (/Nested Loop|Nested loop/i.test(fullText))
    joinStrategy = "Nested Loop";
  else if (/Merge Join/i.test(fullText)) joinStrategy = "Merge Join";

  let estimatedCost: string | undefined;
  const pgCostMatch = fullText.match(/cost=([0-9.]+)\.\.([0-9.]+)/i);
  const mysqlCostMatch = fullText.match(/cost=([0-9.]+)/i);
  if (pgCostMatch) {
    estimatedCost = `${pgCostMatch[1]} .. ${pgCostMatch[2]}`;
  } else if (mysqlCostMatch) {
    estimatedCost = mysqlCostMatch[1];
  }

  let actualTime: string | undefined;
  const execTimeMatch = fullText.match(/Execution Time:\s*([0-9.]+)\s*ms/i);
  const timeMatch =
    fullText.match(/actual time=([0-9.]+)\.\.([0-9.]+)/i) ||
    fullText.match(/time=([0-9.]+)ms/i);
  if (execTimeMatch) {
    actualTime = `${execTimeMatch[1]} ms`;
  } else if (timeMatch) {
    actualTime = timeMatch[2] ? `${timeMatch[2]} ms` : `${timeMatch[1]} ms`;
  }

  let rowsProcessed = mysqlRows;
  if (!rowsProcessed) {
    const rowsMatch = fullText.match(/rows=([0-9]+)/i);
    if (rowsMatch) {
      rowsProcessed = rowsMatch[1];
    }
  }

  return {
    isAnalyze: analyze,
    rawText: fullText || "No plan text returned.",
    scanType,
    indexUsed,
    joinStrategy,
    estimatedCost,
    actualTime,
    rowsProcessed,
  };
}

