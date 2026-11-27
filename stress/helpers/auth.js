import http from 'k6/http';
import exec from 'k6/execution';

export function login(baseURL, username, password) {
  const url = `${baseURL}/api/v1/auth/login`;
  const payload = JSON.stringify({ username, password });
  const params = {
    headers: { 'Content-Type': 'application/json' },
    timeout: '10s',
  };

  const res = http.post(url, payload, params);
  if (res.status !== 200) {
    exec.test.abort(`[Setup] Failed to login as '${username}'. HTTP ${res.status}: ${res.body}`);
  }

  const json = res.json();
  const userData = json.data;
  const sessionCookie = res.cookies.session && res.cookies.session.length > 0 
    ? res.cookies.session[0].value 
    : '';

  if (!sessionCookie) {
    exec.test.abort(`[Setup] Missing session cookie in login response for '${username}'`);
  }

  return {
    id: userData.id,
    username: userData.username,
    role: userData.role,
    headers: {
      'Content-Type': 'application/json',
      Cookie: `session=${sessionCookie}`,
      'X-CSRF-Token': userData.csrfToken,
    },
  };
}
