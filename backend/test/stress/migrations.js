import { sleep } from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';
import { Counter, Rate, Trend } from 'k6/metrics';

const baseURL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const benchmarkRows = Number(__ENV.BENCHMARK_ROWS || 0);
const phases = ['baseline', 'peak', 'recovery'];
const phaseSeconds = 14;
const duration = new Trend('api_duration', true);
const requests = new Counter('api_requests');
const errors = new Rate('api_errors');
const completed = new Counter('jobs_completed');
const conflicts = new Counter('submission_conflicts');
const queueWait = new Trend('queue_wait_per_job', true);
const executionTime = new Trend('execution_per_job', true);
const jobRate = new Trend('jobs_completed_per_second');
const scenarios = benchmarkRows ? {
  benchmark: { executor: 'shared-iterations', vus: 1, iterations: 1, exec: 'benchmark', maxDuration: '4m' },
} : {
  api: {
    executor: 'ramping-vus', exec: 'readAPI', startVUs: 2, gracefulRampDown: '0s',
    stages: [
      { duration: '14s', target: 2 },
      { duration: '4s', target: 12 }, { duration: '10s', target: 12 },
      { duration: '4s', target: 2 }, { duration: '10s', target: 2 },
    ],
  },
  ...Object.fromEntries(phases.map((phase, index) => [phase, {
    executor: 'shared-iterations', vus: 1, iterations: 1, exec: 'submitAndWatch',
    startTime: `${phaseSeconds * index + 5}s`, maxDuration: '8s',
  }])),
};
export const options = {
  scenarios,
  setupTimeout: '2m',
  thresholds: benchmarkRows ? { jobs_completed: ['count==1'] } : Object.fromEntries(
    phases.flatMap((phase) => [
      [`api_duration{phase:${phase}}`, ['p(95)<500']],
      [`api_errors{phase:${phase}}`, ['rate<0.01']],
      [`api_requests{phase:${phase}}`, ['count>0']],
      [`jobs_completed{phase:${phase}}`, ['count==3']],
      [`submission_conflicts{phase:${phase}}`, ['count>=1']],
    ]),
  ),
  summaryTrendStats: ['avg', 'med', 'p(95)'],
};

function data(response, status) {
  if (response.status !== status) {
    exec.test.abort(`Expected HTTP ${status}, got ${response.status} at ${response.url}`);
  }
  return response.json('data');
}

function login(username, password) {
  const response = http.post(`${baseURL}/api/v1/auth/login`, JSON.stringify({ username, password }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const user = data(response, 200);
  return {
    id: user.id,
    headers: {
      'Content-Type': 'application/json',
      Cookie: `session=${response.cookies.session[0].value}`,
      'X-CSRF-Token': user.csrfToken,
    },
  };
}

function api(user, path, body, timeout = '3s') {
  const url = `${baseURL}/api/v1${path}`;
  const params = { headers: user.headers, timeout };
  return body === undefined ? http.get(url, params) : http.post(url, JSON.stringify(body), params);
}

function approveDraft(author, leaders, database, table, expression, batchSize) {
  const draft = data(api(author, `/me/databases/${database.id}/drafts`, {
    sqlUp: `UPDATE ${table} SET ${expression} WHERE id BETWEEN ? AND ?`,
    isBatch: true, batchColumn: 'id', batchSize,
  }), 201);
  const path = `/me/drafts/${draft.id}`;
  for (const id of new Set(draft.approvals.map((approval) => approval.leaderId))) {
    if (!leaders[id]) exec.test.abort(`Missing seeded leader for group vote: ${id}`);
    data(api(leaders[id], `${path}/approvals`, { status: 'approved' }), 200);
  }
  if (data(api(author, path), 200).status !== 'approved') exec.test.abort('Fixture draft was not approved');
  return path;
}

export function setup() {
  const author = login('admin', 'admin123');
  const voters = [author, login('leader', 'leader123'), login('leader_qa', 'qa123'), login('auditor_lead', 'auditor123')];
  const leaders = Object.fromEntries(voters.map((user) => [user.id, user]));
  const databases = data(api(author, '/me/databases'), 200);
  const primary = databases.find((db) => db.name === 'db1_pg');
  const secondary = databases.find((db) => db.name === 'db3_analytics');
  if (!primary || !secondary) exec.test.abort('Seed both PostgreSQL demo targets before running');
  if (benchmarkRows) {
    return { author, benchmark: approveDraft(author, leaders, primary, 'wbd_batch_benchmark', 'score = score + 1', 10000) };
  }
  const jobs = Object.fromEntries(phases.map((phase) => [phase, [
    approveDraft(author, leaders, primary, 'customers', 'name = name', 2),
    approveDraft(author, leaders, primary, 'customers', 'name = name', 2),
    approveDraft(author, leaders, secondary, 'daily_metrics', 'active_users = active_users', 2),
  ]]));
  return { author, jobs };
}

export function readAPI({ author }) {
  const index = Math.min(phases.length - 1, Math.floor(exec.instance.currentTestRunDuration / (phaseSeconds * 1000)));
  const tags = { phase: phases[index] };
  const response = api(author, '/me/databases');
  duration.add(response.timings.duration, tags);
  requests.add(1, tags);
  errors.add(response.status !== 200, tags);
  sleep(0.15);
}

function submit(author, pending, tags) {
  const responseCallback = http.expectedStatuses(202, 409);
  const results = http.batch(pending.map((path) => ({
    method: 'POST', url: `${baseURL}/api/v1${path}/execute`, body: '{}',
    params: { headers: author.headers, timeout: '3s', responseCallback },
  })));
  const accepted = [];
  results.forEach((response, index) => {
    duration.add(response.timings.duration, tags);
    requests.add(1, tags);
    errors.add(response.status !== 202 && response.status !== 409, tags);
    if (response.status === 409) {
      conflicts.add(1, tags);
      return;
    }
    if (!data(response, 202).jobId) exec.test.abort('Accepted request has no job ID');
    accepted.push(pending[index]);
  });
  return accepted;
}

function watch(author, paths, tags, timeoutMs) {
  const start = Date.now();
  const pending = new Set(paths);
  const running = new Set();
  while (pending.size || running.size) {
    if (Date.now() - start >= timeoutMs) exec.test.abort(`Job completion timed out during ${tags.phase}`);
    if (pending.size) {
      for (const path of submit(author, [...pending], tags)) {
        pending.delete(path);
        running.add(path);
      }
    }
    for (const path of running) {
      const poll = api(author, path, undefined, benchmarkRows ? '10s' : '3s');
      if (poll.status === 0) continue;
      const status = data(poll, 200).status;
      if (status === 'applied') {
        running.delete(path);
        completed.add(1, tags);
      } else if (!['approved', 'executing'].includes(status)) {
        exec.test.abort(`Unexpected terminal draft status: ${status}`);
      }
    }
    if (pending.size || running.size) sleep(benchmarkRows ? 1 : 0.2);
  }
  return (Date.now() - start) / 1000;
}

function queueAndExecution() {
  const response = http.get(`${baseURL}/metrics`);
  if (response.status !== 200) exec.test.abort('Application metrics are unavailable');
  const read = (key) => Number(response.body.split('\n').find((line) => line.startsWith(key + ' '))?.split(' ')[1] || 0);
  return [
    read('migration_queue_wait_duration_seconds_sum'),
    read('migration_execution_duration_seconds_sum{operation="execute",status="success"}'),
  ];
}

function measureJobs(author, paths, phase, timeoutMs) {
  const tags = { phase };
  const before = queueAndExecution();
  const elapsed = watch(author, paths, tags, timeoutMs);
  const after = queueAndExecution();
  queueWait.add((after[0] - before[0]) * 1000 / paths.length, tags);
  executionTime.add((after[1] - before[1]) * 1000 / paths.length, tags);
  jobRate.add(paths.length / elapsed, tags);
  console.log(JSON.stringify({
    phase, completed: paths.length, elapsedSeconds: elapsed,
    queueWaitMs: (after[0] - before[0]) * 1000 / paths.length,
    executionMs: (after[1] - before[1]) * 1000 / paths.length,
    jobsPerSecond: paths.length / elapsed,
  }));
}

export function submitAndWatch({ author, jobs }) {
  const phase = exec.scenario.name;
  measureJobs(author, jobs[phase], phase, 7000);
}

export function benchmark({ author, benchmark: path }) {
  measureJobs(author, [path], 'benchmark', 220000);
  const progress = data(api(author, `${path}/progress`, undefined, '10s'), 200);
  if (progress.rows_processed !== benchmarkRows) {
    exec.test.abort(`Expected ${benchmarkRows} committed rows, got ${progress.rows_processed}`);
  }
  console.log(JSON.stringify({ committedRows: progress.rows_processed, batches: progress.current_batch }));
}
