import http from 'k6/http';
import exec from 'k6/execution';

export function createAndApproveDraft(baseURL, author, leadersMap, databaseID, draftPayload) {
  const createUrl = `${baseURL}/api/v1/me/databases/${databaseID}/drafts`;
  const createRes = http.post(createUrl, JSON.stringify(draftPayload), {
    headers: author.headers,
    timeout: '10s',
  });

  if (createRes.status !== 201) {
    exec.test.abort(`[Draft Factory] Failed to create draft on DB ${databaseID}. HTTP ${createRes.status}: ${createRes.body}`);
  }

  const draft = createRes.json('data');
  const draftID = draft.id;
  const approvals = draft.approvals || [];

  const requiredLeaders = new Set(approvals.map((a) => a.leaderId).filter(Boolean));
  for (const leaderID of requiredLeaders) {
    const leaderUser = leadersMap[leaderID];
    if (!leaderUser) {
      exec.test.abort(`[Draft Factory] Missing authenticated leader user for ID ${leaderID}`);
    }

    const voteUrl = `${baseURL}/api/v1/me/drafts/${draftID}/approvals`;
    const voteRes = http.post(voteUrl, JSON.stringify({ status: 'approved' }), {
      headers: leaderUser.headers,
      timeout: '10s',
    });

    if (voteRes.status !== 200) {
      exec.test.abort(`[Draft Factory] Leader ${leaderID} failed to approve draft ${draftID}. HTTP ${voteRes.status}: ${voteRes.body}`);
    }
  }

  const verifyUrl = `${baseURL}/api/v1/me/drafts/${draftID}`;
  const verifyRes = http.get(verifyUrl, { headers: author.headers, timeout: '5s' });
  const finalStatus = verifyRes.json('data.status');
  if (finalStatus !== 'approved') {
    exec.test.abort(`[Draft Factory] Draft ${draftID} expected 'approved', but found '${finalStatus}'`);
  }

  return {
    id: draftID,
    databaseID,
    path: `/me/drafts/${draftID}`,
    isBatch: draftPayload.isBatch || false,
  };
}

export function provisionDraftBatch(baseURL, author, leadersMap, databaseID, count, templateFunc) {
  const drafts = [];
  for (let i = 0; i < count; i++) {
    const payload = templateFunc(i);
    const draft = createAndApproveDraft(baseURL, author, leadersMap, databaseID, payload);
    drafts.push(draft);
  }
  return drafts;
}
