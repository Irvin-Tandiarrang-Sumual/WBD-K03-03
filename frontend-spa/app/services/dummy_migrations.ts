import type { MigrationRecord } from '../types';
import { updateDatabaseHeadVersion } from './dummy_databases';

export const INITIAL_MIGRATIONS: MigrationRecord[] = [
  {
    id: 'mig-p5-fail',
    databaseId: 'db-payments',
    version: 5,
    summary: 'Widen settlements_ledger.currency_code',
    type: 'UP',
    status: 'Fail',
    reversible: true,
    executedBy: 'leader_alice',
    executedAtRelative: '2h ago',
    durationMs: 412,
    upSql: `ALTER TABLE settlements_ledger\n  ALTER COLUMN currency_code TYPE VARCHAR(8);\n\nCREATE INDEX CONCURRENTLY idx_settlements_holiday_batch\n  ON settlements_ledger (settlement_date, currency_code);`,
    downSql: `-- System-generated rollback\nDROP INDEX CONCURRENTLY IF EXISTS idx_settlements_holiday_batch;\n\nALTER TABLE settlements_ledger\n  ALTER COLUMN currency_code TYPE CHAR(3);`,
    errorLog: `ERROR: cannot alter type of a column used by a view or rule (SQLSTATE 0A000)\nDETAIL: rule _RETURN on view v_daily_settlement_rollup depends on column "currency_code"`,
    diffFromVersion: 4,
    schemaDiff: [
      { type: 'header', content: '@@ settlements_ledger (#4 → #5) @@' },
      { type: 'context', content: '  settlements_ledger.id BIGINT PRIMARY KEY' },
      { type: 'remove', content: '- settlements_ledger.currency_code CHAR(3) NOT NULL' },
      { type: 'add', content: '+ settlements_ledger.currency_code VARCHAR(8) NOT NULL' },
      { type: 'add', content: '+ INDEX idx_settlements_holiday_batch (settlement_date, currency_code)' },
    ],
  },
  {
    id: 'mig-p4',
    databaseId: 'db-payments',
    version: 4,
    summary: 'Add users.phone, drop legacy_field',
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy: 'leader_alice',
    executedAtRelative: '2d ago',
    durationMs: 184,
    upSql: `ALTER TABLE users\n  ADD COLUMN phone VARCHAR(20),\n  DROP COLUMN legacy_field;\n\nCREATE INDEX idx_users_phone ON users (phone) WHERE phone IS NOT NULL;`,
    downSql: `-- System-generated rollback\nDROP INDEX IF EXISTS idx_users_phone;\n\nALTER TABLE users\n  DROP COLUMN phone,\n  ADD COLUMN legacy_field INT DEFAULT 0;`,
    diffFromVersion: 3,
    schemaDiff: [
      { type: 'header', content: '@@ users (#3 → #4) @@' },
      { type: 'context', content: '  users.id UUID PRIMARY KEY' },
      { type: 'context', content: '  users.email VARCHAR(255) NOT NULL UNIQUE' },
      { type: 'add', content: '+ users.phone VARCHAR(20)' },
      { type: 'remove', content: '- users.legacy_field INT' },
      { type: 'add', content: '+ INDEX idx_users_phone (phone)' },
    ],
  },
  {
    id: 'mig-p3-rb',
    databaseId: 'db-payments',
    version: 3,
    summary: 'Revert payout_batches.retry_metadata',
    type: 'ROLLBACK',
    status: 'Success',
    reversible: true,
    executedBy: 'devon.park',
    executedAtRelative: '5d ago',
    durationMs: 92,
    upSql: `ALTER TABLE payout_batches ADD COLUMN retry_metadata JSONB DEFAULT '{}'::jsonb;`,
    downSql: `-- System-generated rollback\nALTER TABLE payout_batches DROP COLUMN IF EXISTS retry_metadata;`,
    diffFromVersion: 3,
    schemaDiff: [
      { type: 'header', content: '@@ payout_batches (Rollback #3 → #2) @@' },
      { type: 'remove', content: '- payout_batches.retry_metadata JSONB' },
    ],
  },
  {
    id: 'mig-p3',
    databaseId: 'db-payments',
    version: 3,
    summary: 'Add payout_batches.retry_metadata',
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy: 'irvin',
    executedAtRelative: '5d ago',
    durationMs: 126,
    upSql: `ALTER TABLE payout_batches\n  ADD COLUMN retry_metadata JSONB DEFAULT '{}'::jsonb NOT NULL;`,
    downSql: `-- System-generated rollback\nALTER TABLE payout_batches\n  DROP COLUMN IF EXISTS retry_metadata;`,
    diffFromVersion: 2,
    schemaDiff: [
      { type: 'header', content: '@@ payout_batches (#2 → #3) @@' },
      { type: 'context', content: '  payout_batches.id BIGSERIAL PRIMARY KEY' },
      { type: 'add', content: '+ payout_batches.retry_metadata JSONB NOT NULL' },
    ],
  },
  {
    id: 'mig-p2',
    databaseId: 'db-payments',
    version: 2,
    summary: 'Drop legacy_card_vault_v1',
    type: 'UP',
    status: 'Success',
    reversible: false,
    executedBy: 'admin',
    executedAtRelative: '12d ago',
    durationMs: 638,
    upSql: `DROP TABLE IF EXISTS legacy_card_vault_v1 CASCADE;\nUPDATE schema_audit_markers SET pci_scope = 'v4_tokenized' WHERE active = true;`,
    diffFromVersion: 1,
    schemaDiff: [
      { type: 'header', content: '@@ public (#1 → #2) @@' },
      { type: 'remove', content: '- TABLE legacy_card_vault_v1 (id, pan_hash, exp_month, exp_year)' },
      { type: 'context', content: '  TABLE schema_audit_markers' },
    ],
  },
  {
    id: 'mig-l3',
    databaseId: 'db-ledger',
    version: 3,
    summary: 'Add voucher balance columns',
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy: 'irvin',
    executedAtRelative: '1d ago',
    durationMs: 142,
    upSql: `ALTER TABLE ledger_accounts\n  ADD COLUMN voucher_balance_cents BIGINT NOT NULL DEFAULT 0,\n  ADD COLUMN voucher_expires_at TIMESTAMPTZ;\n\nCREATE INDEX idx_ledger_voucher_expiry\n  ON ledger_accounts (voucher_expires_at) WHERE voucher_balance_cents > 0;`,
    downSql: `-- System-generated rollback\nDROP INDEX IF EXISTS idx_ledger_voucher_expiry;\n\nALTER TABLE ledger_accounts\n  DROP COLUMN IF EXISTS voucher_expires_at,\n  DROP COLUMN IF EXISTS voucher_balance_cents;`,
    diffFromVersion: 2,
    schemaDiff: [
      { type: 'header', content: '@@ ledger_accounts (#2 → #3) @@' },
      { type: 'context', content: '  ledger_accounts.account_id UUID PRIMARY KEY' },
      { type: 'context', content: '  ledger_accounts.balance_cents BIGINT NOT NULL' },
      { type: 'add', content: '+ ledger_accounts.voucher_balance_cents BIGINT NOT NULL DEFAULT 0' },
      { type: 'add', content: '+ ledger_accounts.voucher_expires_at TIMESTAMPTZ' },
      { type: 'add', content: '+ INDEX idx_ledger_voucher_expiry (voucher_expires_at)' },
    ],
  },
  {
    id: 'mig-l2',
    databaseId: 'db-ledger',
    version: 2,
    summary: 'Add users.phone, drop legacy_field',
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy: 'hannah.okafor',
    executedAtRelative: '6d ago',
    durationMs: 118,
    upSql: `ALTER TABLE users\n  ADD COLUMN phone VARCHAR(20),\n  DROP COLUMN legacy_field;`,
    downSql: `-- System-generated rollback\nALTER TABLE users\n  DROP COLUMN phone,\n  ADD COLUMN legacy_field INT;`,
    diffFromVersion: 1,
    schemaDiff: [
      { type: 'header', content: '@@ users (#1 → #2) @@' },
      { type: 'context', content: '  users.id UUID PRIMARY KEY' },
      { type: 'add', content: '+ users.phone VARCHAR(20)' },
      { type: 'remove', content: '- users.legacy_field INT' },
    ],
  },
  {
    id: 'mig-l1-rb',
    databaseId: 'db-ledger',
    version: 1,
    summary: 'Revert Q4 journal partition',
    type: 'ROLLBACK',
    status: 'Success',
    reversible: true,
    executedBy: 'leader_alice',
    executedAtRelative: '10d ago',
    durationMs: 211,
    upSql: `CREATE TABLE journal_entries_2026_q4 PARTITION OF journal_entries FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');`,
    downSql: `-- System-generated rollback\nDROP TABLE IF EXISTS journal_entries_2026_q4;`,
    diffFromVersion: 1,
    schemaDiff: [
      { type: 'header', content: '@@ partition rollback (#1 → #0) @@' },
      { type: 'remove', content: '- TABLE journal_entries_2026_q4' },
    ],
  },
  {
    id: 'mig-l1',
    databaseId: 'db-ledger',
    version: 1,
    summary: 'Create journal_entries table',
    type: 'UP',
    status: 'Success',
    reversible: false,
    executedBy: 'admin',
    executedAtRelative: '18d ago',
    durationMs: 305,
    upSql: `CREATE TABLE journal_entries (\n  entry_id BIGSERIAL PRIMARY KEY,\n  account_id UUID NOT NULL REFERENCES ledger_accounts(account_id),\n  delta_cents BIGINT NOT NULL,\n  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);`,
    diffFromVersion: 0,
    schemaDiff: [
      { type: 'header', content: '@@ journal_entries (#0 → #1) @@' },
      { type: 'add', content: '+ TABLE journal_entries' },
      { type: 'add', content: '+ journal_entries.entry_id BIGSERIAL PRIMARY KEY' },
      { type: 'add', content: '+ journal_entries.account_id UUID NOT NULL' },
      { type: 'add', content: '+ journal_entries.delta_cents BIGINT NOT NULL' },
    ],
  },
];

let memoryMigrations: MigrationRecord[] = [...INITIAL_MIGRATIONS];

export const fetchMigrationsByDatabaseId = async (databaseId: string): Promise<MigrationRecord[]> => {
  await new Promise((resolve) => setTimeout(resolve, 80));
  return memoryMigrations.filter((m) => m.databaseId === databaseId);
};

export const fetchMigrationById = async (id: string): Promise<MigrationRecord | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 50));
  return memoryMigrations.find((m) => m.id === id);
};

export const appendMigration = async (record: Omit<MigrationRecord, 'id'> & { id?: string }): Promise<MigrationRecord> => {
  await new Promise((resolve) => setTimeout(resolve, 100));
  const newRecord: MigrationRecord = {
    ...record,
    id: record.id || `mig-${Date.now()}`,
    durationMs: record.durationMs || 120,
    summary: record.summary || 'Applied migration',
  };
  memoryMigrations = [newRecord, ...memoryMigrations];
  return newRecord;
};

export const executeRollback = async (
  databaseId: string,
  executedBy: string
): Promise<{ success: boolean; newHeadVersion: number }> => {
  await new Promise((resolve) => setTimeout(resolve, 150));
  const currentDbMigrations = memoryMigrations
    .filter((m) => m.databaseId === databaseId && m.status === 'Success' && m.type === 'UP')
    .sort((a, b) => b.version - a.version);

  const headMig = currentDbMigrations[0];
  if (!headMig || !headMig.reversible) {
    return { success: false, newHeadVersion: headMig ? headMig.version : 0 };
  }

  const nextHeadVersion = headMig.version - 1;
  const rollbackRecord: MigrationRecord = {
    id: `mig-rb-${Date.now()}`,
    databaseId,
    version: nextHeadVersion,
    summary: `Rollback: ${headMig.summary}`,
    type: 'ROLLBACK',
    status: 'Success',
    reversible: true,
    executedBy,
    executedAtRelative: 'Just now',
    durationMs: 88,
    upSql: headMig.downSql || '-- Rollback completed',
    downSql: headMig.upSql,
    diffFromVersion: headMig.version,
    schemaDiff: [
      { type: 'header', content: `@@ ${headMig.summary} (Rollback to #${nextHeadVersion}) @@` },
      { type: 'remove', content: `- Reverted changes from version #${headMig.version}` },
    ],
  };

  memoryMigrations = [rollbackRecord, ...memoryMigrations];
  updateDatabaseHeadVersion(databaseId, nextHeadVersion, [headMig.version]);
  return { success: true, newHeadVersion: nextHeadVersion };
};

export const executeForwardRedo = async (
  databaseId: string,
  targetVersion: number,
  executedBy: string
): Promise<{ success: boolean; newHeadVersion: number }> => {
  await new Promise((resolve) => setTimeout(resolve, 150));
  const redoRecord: MigrationRecord = {
    id: `mig-fwd-${Date.now()}`,
    databaseId,
    version: targetVersion,
    summary: `Redo forward to version #${targetVersion}`,
    type: 'UP',
    status: 'Success',
    reversible: true,
    executedBy,
    executedAtRelative: 'Just now',
    durationMs: 110,
    upSql: `-- Redo forward to version ${targetVersion}`,
    diffFromVersion: targetVersion - 1,
    schemaDiff: [
      { type: 'header', content: `@@ Reapplied Version #${targetVersion} @@` },
      { type: 'add', content: `+ Forward redo executed successfully` },
    ],
  };

  memoryMigrations = [redoRecord, ...memoryMigrations];
  updateDatabaseHeadVersion(databaseId, targetVersion, []);
  return { success: true, newHeadVersion: targetVersion };
};
