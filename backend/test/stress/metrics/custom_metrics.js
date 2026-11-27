import { Counter, Rate, Trend } from 'k6/metrics';

export const apiDuration = new Trend('api_duration', true);
export const apiRequests = new Counter('api_requests');
export const apiUnexpectedErrors = new Rate('api_unexpected_errors');
export const lockConflicts = new Counter('lock_conflicts');
export const quarantineLocks = new Counter('quarantine_locks');

export const migrationJobsSubmitted = new Counter('migration_jobs_submitted');
export const migrationJobsApplied = new Counter('migration_jobs_applied');
export const migrationJobsFailed = new Counter('migration_jobs_failed');
export const migrationJobSuccessRate = new Rate('migration_job_success_rate');

export const migrationQueueWaitMs = new Trend('migration_queue_wait_ms', true);
export const migrationExecutionMs = new Trend('migration_execution_ms', true);
export const migrationJobTotalMs = new Trend('migration_job_total_ms', true);

export const recoveryApiLatency = new Trend('recovery_api_latency', true);
export const poolExhaustionErrors = new Rate('pool_exhaustion_errors');
