const isQuick = __ENV.STRESS_PROFILE === 'quick';

export const config = {
  baseURL: (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, ''),
  isQuick,

  users: {
    admin: { username: 'admin', password: 'admin123' },
    leadEng: { username: 'leader', password: 'leader123' },
    leadQA: { username: 'leader_qa', password: 'qa123' },
    leadAuditor: { username: 'auditor_lead', password: 'auditor123' },
    regularDev: { username: 'user', password: 'user123' },
  },

  databases: {
    primary: 'db1_pg',
    secondary: 'db3_analytics',
    billing: 'db4_billing',
    mysql: 'db2_mysql',
  },

  stages: isQuick
    ? [
        { duration: '10s', target: 5 },
        { duration: '20s', target: 20 },
        { duration: '15s', target: 20 },
        { duration: '10s', target: 5 },
        { duration: '5s', target: 0 },
      ]
    : [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 30 },
        { duration: '1m', target: 75 },
        { duration: '30s', target: 75 },
        { duration: '30s', target: 10 },
        { duration: '45s', target: 0 },
      ],

  thresholds: {
    'http_req_duration{type:api}': ['p(95)<800', 'p(99)<2000'],
    'api_unexpected_errors': ['rate<0.03'],
    'lock_conflicts': ['count>=0'],
    'migration_queue_wait_ms': ['p(95)<5000'],
    'migration_job_success_rate': ['rate>0.95'],
  },

  jobPollTimeoutMs: Number(__ENV.JOB_POLL_TIMEOUT_MS || 60000),
  jobPollIntervalMs: Number(__ENV.JOB_POLL_INTERVAL_MS || 300),
};
