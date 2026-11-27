import { sleep } from 'k6';
import { config } from '../config/environments.js';
import { request } from '../helpers/api_client.js';
import { pollDraftUntilTerminal } from '../helpers/job_poller.js';
import {
  migrationJobsSubmitted,
  lockConflicts,
} from '../metrics/custom_metrics.js';

export function runCrossDbSaturation(data, currentPhase, crossDraftPools) {
  const { author, baseURL } = data;
  const tags = { scenario: 'cross_db_saturation', phase: currentPhase };

  const availableDBs = Object.keys(crossDraftPools).filter(
    (dbKey) => crossDraftPools[dbKey] && crossDraftPools[dbKey].length > 0
  );

  if (availableDBs.length === 0) {
    sleep(0.5);
    return;
  }

  const chosenDBKey = availableDBs[Math.floor(Math.random() * availableDBs.length)];
  const draft = crossDraftPools[chosenDBKey].pop();
  if (!draft) return;

  const targetTags = Object.assign({}, tags, { db: chosenDBKey });
  const execPath = `/me/drafts/${draft.id}/execute`;
  const res = request('POST', baseURL, author, execPath, {}, targetTags);

  if (res.status === 202) {
    migrationJobsSubmitted.add(1, targetTags);
    pollDraftUntilTerminal(baseURL, author, draft.id, config.jobPollTimeoutMs, config.jobPollIntervalMs, targetTags);
  } else if (res.status === 409) {
    lockConflicts.add(1, targetTags);
    sleep(0.3);
  }
}
