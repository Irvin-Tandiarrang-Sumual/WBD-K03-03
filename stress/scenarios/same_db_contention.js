import { sleep } from 'k6';
import { config } from '../config/environments.js';
import { request } from '../helpers/api_client.js';
import { pollDraftUntilTerminal } from '../helpers/job_poller.js';
import {
  migrationJobsSubmitted,
  lockConflicts,
} from '../metrics/custom_metrics.js';

export function runSameDbContention(data, currentPhase, draftPool) {
  const { author, primaryDB, baseURL } = data;
  const tags = { scenario: 'same_db_contention', target: primaryDB.name, phase: currentPhase };

  if (!draftPool || draftPool.length === 0) {
    sleep(0.5);
    return;
  }

  const draft = draftPool.pop();
  if (!draft) return;

  const execPath = `/me/drafts/${draft.id}/execute`;
  const res = request('POST', baseURL, author, execPath, {}, tags);

  if (res.status === 202) {
    migrationJobsSubmitted.add(1, tags);
    pollDraftUntilTerminal(baseURL, author, draft.id, config.jobPollTimeoutMs, config.jobPollIntervalMs, tags);
  } else if (res.status === 409) {
    lockConflicts.add(1, tags);
    sleep(0.5);
  }
}
