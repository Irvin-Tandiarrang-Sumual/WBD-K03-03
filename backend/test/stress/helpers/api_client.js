import http from 'k6/http';
import {
  apiDuration,
  apiRequests,
  apiUnexpectedErrors,
  lockConflicts,
  quarantineLocks,
  poolExhaustionErrors,
} from '../metrics/custom_metrics.js';

export function request(method, baseURL, user, path, body = null, tags = {}, timeout = '8s') {
  const url = `${baseURL}/api/v1${path}`;
  const metricTags = Object.assign({ type: 'api', path }, tags);
  const params = {
    headers: user.headers,
    timeout,
    tags: metricTags,
  };

  const payload = body !== null ? JSON.stringify(body) : null;
  const res = method === 'GET' ? http.get(url, params) : http.post(url, payload, params);

  apiRequests.add(1, metricTags);
  if (res.timings && res.timings.duration > 0) {
    apiDuration.add(res.timings.duration, metricTags);
  }

  if (res.status === 409) {
    lockConflicts.add(1, metricTags);
  } else if (res.status === 423) {
    quarantineLocks.add(1, metricTags);
  } else if (res.status >= 500) {
    apiUnexpectedErrors.add(1, metricTags);
    if (res.body && (res.body.includes('pool') || res.body.includes('connection') || res.body.includes('too many clients'))) {
      poolExhaustionErrors.add(1, metricTags);
    }
  } else if (res.status >= 400 && res.status !== 401 && res.status !== 404) {
    apiUnexpectedErrors.add(1, metricTags);
  } else {
    apiUnexpectedErrors.add(0, metricTags);
  }

  return res;
}
