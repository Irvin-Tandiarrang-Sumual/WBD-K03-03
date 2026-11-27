import http from 'k6/http';
import { sleep } from 'k6';
import {
  migrationJobTotalMs,
  migrationJobsApplied,
  migrationJobsFailed,
  migrationJobSuccessRate,
} from '../metrics/custom_metrics.js';

export function pollDraftUntilTerminal(baseURL, user, draftID, timeoutMs = 45000, intervalMs = 300, tags = {}) {
  const start = Date.now();
  const url = `${baseURL}/api/v1/me/drafts/${draftID}`;
  const metricTags = Object.assign({ type: 'job_poll' }, tags);
  const params = {
    headers: user.headers,
    timeout: '5s',
    tags: metricTags,
  };

  while (Date.now() - start < timeoutMs) {
    const res = http.get(url, params);
    if (res.status === 200) {
      const json = res.json();
      const status = json.data ? json.data.status : '';

      if (status === 'applied') {
        const totalDuration = Date.now() - start;
        migrationJobTotalMs.add(totalDuration, metricTags);
        migrationJobsApplied.add(1, metricTags);
        migrationJobSuccessRate.add(1, metricTags);
        return { success: true, status: 'applied', durationMs: totalDuration };
      }

      if (status === 'failed' || status === 'partial' || status === 'partially_reverted') {
        const totalDuration = Date.now() - start;
        migrationJobTotalMs.add(totalDuration, metricTags);
        migrationJobsFailed.add(1, metricTags);
        migrationJobSuccessRate.add(0, metricTags);
        return { success: false, status, durationMs: totalDuration, error: `Terminal state: ${status}` };
      }
    }

    sleep(intervalMs / 1000);
  }

  const totalDuration = Date.now() - start;
  migrationJobTotalMs.add(totalDuration, metricTags);
  migrationJobsFailed.add(1, metricTags);
  migrationJobSuccessRate.add(0, metricTags);
  return { success: false, status: 'timeout', durationMs: totalDuration, error: `Polling exceeded ${timeoutMs}ms timeout` };
}
