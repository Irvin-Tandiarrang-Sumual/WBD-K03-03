import exec from 'k6/execution';
import { config } from './config/environments.js';
import { login } from './helpers/auth.js';
import { request } from './helpers/api_client.js';
import { provisionDraftBatch } from './helpers/draft_factory.js';
import { runApiTraffic } from './scenarios/api_traffic.js';
import { runSameDbContention } from './scenarios/same_db_contention.js';
import { runCrossDbSaturation } from './scenarios/cross_db_saturation.js';
import { recoveryApiLatency } from './metrics/custom_metrics.js';

export const options = {
  scenarios: {
    api_stress: {
      executor: 'ramping-vus',
      exec: 'apiStressRunner',
      startVUs: 2,
      stages: config.stages,
      gracefulRampDown: '5s',
    },
    same_db_contention: {
      executor: 'ramping-vus',
      exec: 'sameDbRunner',
      startVUs: 1,
      stages: config.isQuick
        ? [
            { duration: '10s', target: 2 },
            { duration: '20s', target: 5 },
            { duration: '20s', target: 5 },
            { duration: '10s', target: 0 },
          ]
        : [
            { duration: '30s', target: 2 },
            { duration: '1m',  target: 5 },
            { duration: '1m',  target: 10 },
            { duration: '30s', target: 10 },
            { duration: '30s', target: 1 },
            { duration: '45s', target: 0 },
          ],
      gracefulRampDown: '5s',
    },
    cross_db_saturation: {
      executor: 'ramping-vus',
      exec: 'crossDbRunner',
      startVUs: 1,
      stages: config.isQuick
        ? [
            { duration: '10s', target: 2 },
            { duration: '20s', target: 4 },
            { duration: '20s', target: 4 },
            { duration: '10s', target: 0 },
          ]
        : [
            { duration: '30s', target: 2 },
            { duration: '1m',  target: 4 },
            { duration: '1m',  target: 8 },
            { duration: '30s', target: 8 },
            { duration: '30s', target: 1 },
            { duration: '45s', target: 0 },
          ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    'api_duration{type:api}': ['p(95)<800', 'p(99)<2000'],
    'api_unexpected_errors': [{ threshold: 'rate<0.03', abortOnFail: true, delayAbortEval: '30s' }],
    'migration_queue_wait_ms': ['p(95)<8000'],
    'migration_job_success_rate': ['rate>0.90'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

let sameDbPool = [];
let crossDbPools = {};

export function setup() {
  const baseURL = config.baseURL;

  const author = login(baseURL, config.users.admin.username, config.users.admin.password);
  const leadEng = login(baseURL, config.users.leadEng.username, config.users.leadEng.password);
  const leadQA = login(baseURL, config.users.leadQA.username, config.users.leadQA.password);
  const leadAuditor = login(baseURL, config.users.leadAuditor.username, config.users.leadAuditor.password);

  const leadersMap = {
    [author.id]: author,
    [leadEng.id]: leadEng,
    [leadQA.id]: leadQA,
    [leadAuditor.id]: leadAuditor,
  };

  const dbListRes = request('GET', baseURL, author, '/me/databases');
  if (dbListRes.status !== 200) {
    exec.test.abort(`[k6 Setup] Cannot list databases: HTTP ${dbListRes.status}`);
  }
  const databases = dbListRes.json('data') || [];
  const primaryDB = databases.find((d) => d.name === config.databases.primary);
  const secondaryDB = databases.find((d) => d.name === config.databases.secondary);
  const billingDB = databases.find((d) => d.name === config.databases.billing);

  if (!primaryDB || !secondaryDB) {
    exec.test.abort(`[k6 Setup] Required demo targets (${config.databases.primary}, ${config.databases.secondary}) not found.`);
  }

  const sameBatchCount = config.isQuick ? 10 : 25;
  const crossBatchCount = config.isQuick ? 6 : 15;

  const sameDbDrafts = provisionDraftBatch(baseURL, author, leadersMap, primaryDB.id, sameBatchCount, () => ({
    sqlUp: `UPDATE customers SET name = name WHERE id BETWEEN ? AND ?`,
    sqlDown: `UPDATE customers SET name = name WHERE id BETWEEN ? AND ?`,
    isBatch: true,
    batchColumn: 'id',
    batchSize: 50,
  }));

  const crossDbDrafts = {
    [primaryDB.name]: provisionDraftBatch(baseURL, author, leadersMap, primaryDB.id, crossBatchCount, () => ({
      sqlUp: `UPDATE orders SET status = status WHERE id BETWEEN ? AND ?`,
      isBatch: true,
      batchColumn: 'id',
      batchSize: 100,
    })),
    [secondaryDB.name]: provisionDraftBatch(baseURL, author, leadersMap, secondaryDB.id, crossBatchCount, () => ({
      sqlUp: `UPDATE daily_metrics SET active_users = active_users WHERE id BETWEEN ? AND ?`,
      isBatch: true,
      batchColumn: 'id',
      batchSize: 50,
    })),
  };

  if (billingDB) {
    const billingBatchCount = config.isQuick ? 4 : 10;
    crossDbDrafts[billingDB.name] = provisionDraftBatch(baseURL, author, leadersMap, billingDB.id, billingBatchCount, () => ({
      sqlUp: `UPDATE invoices SET amount = amount WHERE id BETWEEN ? AND ?`,
      isBatch: true,
      batchColumn: 'id',
      batchSize: 200,
    }));
  }

  return {
    baseURL,
    author,
    primaryDB,
    secondaryDB,
    billingDB,
    sameDbDrafts,
    crossDbDrafts,
  };
}

function resolvePhase() {
  const elapsed = exec.instance.currentTestRunDuration;
  if (config.isQuick) {
    if (elapsed < 10000) return 'normal_baseline';
    if (elapsed < 30000) return 'above_normal';
    if (elapsed < 45000) return 'peak_stress';
    if (elapsed < 55000) return 'ramp_down';
    return 'recovery';
  }
  if (elapsed < 30000) return 'normal_baseline';
  if (elapsed < 90000) return 'above_normal';
  if (elapsed < 150000) return 'peak_stress';
  if (elapsed < 180000) return 'peak_soak';
  if (elapsed < 210000) return 'ramp_down';
  return 'recovery';
}

export function apiStressRunner(data) {
  const phase = resolvePhase();
  runApiTraffic(data, phase);
}

export function sameDbRunner(data) {
  if (sameDbPool.length === 0 && data.sameDbDrafts) {
    sameDbPool = data.sameDbDrafts.slice();
  }
  const phase = resolvePhase();
  runSameDbContention(data, phase, sameDbPool);
}

export function crossDbRunner(data) {
  if (Object.keys(crossDbPools).length === 0 && data.crossDbDrafts) {
    crossDbPools = {};
    for (const k of Object.keys(data.crossDbDrafts)) {
      crossDbPools[k] = data.crossDbDrafts[k].slice();
    }
  }
  const phase = resolvePhase();
  runCrossDbSaturation(data, phase, crossDbPools);
}

export function teardown(data) {
  const { baseURL, author, primaryDB } = data;
  const start = Date.now();
  const res = request('GET', baseURL, author, `/me/databases/${primaryDB.id}`, null, { phase: 'post_stress_probe' });
  const probeDuration = (res && res.timings && res.timings.duration > 0) ? res.timings.duration : (Date.now() - start);
  recoveryApiLatency.add(probeDuration);
}
