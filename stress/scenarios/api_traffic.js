import { sleep } from 'k6';
import { request } from '../helpers/api_client.js';
import {
  generateRandomReadQuery,
  generateExplainAnalyzeQuery,
  randomInt,
} from '../data/payload_generator.js';

export function runApiTraffic(data, currentPhase) {
  const { author, primaryDB, secondaryDB, baseURL } = data;
  const tags = { scenario: 'api_traffic', phase: currentPhase };

  const rand = randomInt(1, 100);

  if (rand <= 40) {
    request('GET', baseURL, author, '/me/databases', null, tags);
    request('GET', baseURL, author, `/me/databases/${primaryDB.id}`, null, tags);
  } else if (rand <= 70) {
    const sql = generateRandomReadQuery();
    request('POST', baseURL, author, `/me/databases/${primaryDB.id}/query`, { query: sql }, tags);
  } else if (rand <= 90) {
    const hideRollback = rand % 2 === 0;
    request('GET', baseURL, author, `/me/databases/${primaryDB.id}/migrations?hide_rollback=${hideRollback}`, null, tags);
  } else {
    const explainSql = generateExplainAnalyzeQuery();
    request('POST', baseURL, author, `/me/databases/${secondaryDB.id}/explain-analyze`, { query: explainSql }, tags);
  }

  sleep(randomInt(50, 200) / 1000);
}
