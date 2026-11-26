export interface SchemaDiffLine {
  type: 'header' | 'add' | 'remove' | 'context';
  content: string;
}

export interface MigrationRecord {
  id: string;
  databaseId: string;
  version: number;
  summary: string;
  type: 'UP' | 'ROLLBACK';
  status: 'Success' | 'Fail';
  reversible: boolean;
  executedBy: string;
  executedAt?: string;
  executedAtRelative: string;
  durationMs: number;
  upSql: string;
  downSql?: string;
  errorLog?: string;
  diffFromVersion: number;
  schemaDiff: SchemaDiffLine[];
}

export type MigrationDirectionFilter = 'all' | 'UP' | 'ROLLBACK';
export type MigrationStatusFilter = 'all' | 'Success' | 'Fail';
