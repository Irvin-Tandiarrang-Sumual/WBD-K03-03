CREATE DATABASE db3_analytics;
\c db3_analytics;

CREATE TABLE daily_metrics (
    id BIGSERIAL PRIMARY KEY,
    metric_date DATE NOT NULL,
    active_users INT NOT NULL DEFAULT 0,
    total_sales NUMERIC(14, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor VARCHAR(80) NOT NULL,
    action VARCHAR(120) NOT NULL,
    details TEXT,
    logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO daily_metrics (metric_date, active_users, total_sales) VALUES
    ('2026-11-01', 110, 14200000.00),
    ('2026-11-02', 125, 16800000.00),
    ('2026-11-03', 140, 19500000.00),
    ('2026-11-04', 135, 17200000.00),
    ('2026-11-05', 158, 22100000.00),
    ('2026-11-06', 170, 24500000.00),
    ('2026-11-07', 162, 21800000.00),
    ('2026-11-08', 180, 26400000.00),
    ('2026-11-09', 195, 28900000.00),
    ('2026-11-10', 210, 31500000.00),
    ('2026-11-11', 205, 29800000.00),
    ('2026-11-12', 225, 34200000.00),
    ('2026-11-13', 240, 38100000.00),
    ('2026-11-14', 230, 36500000.00),
    ('2026-11-15', 250, 41200000.00);

INSERT INTO audit_logs (actor, action, details) VALUES
    ('system', 'DAILY_ROLLUP', 'Daily aggregation completed for 2026-11-01'),
    ('admin', 'CONFIG_SYNC', 'Database pool max connections set to 50'),
    ('user', 'SCHEMA_CHECK', 'Health check passed for analytical pipelines'),
    ('system', 'CLEANUP_JOB', 'Expired cache entries purged (4,210 keys)'),
    ('admin', 'ACCESS_GRANT', 'Read access granted to analytics staging pipeline'),
    ('system', 'DAILY_ROLLUP', 'Daily aggregation completed for 2026-11-15');

INSERT INTO daily_metrics (metric_date, active_users, total_sales)
SELECT
    (CURRENT_DATE - (id || ' days')::interval)::date,
    (100 + (id * 7 % 2000)),
    ((id * 43 % 50000000) + 5000000)::numeric(14,2)
FROM generate_series(16, 5000) AS id;

INSERT INTO audit_logs (actor, action, details)
SELECT
    CASE (id % 4)
        WHEN 0 THEN 'system'
        WHEN 1 THEN 'admin'
        WHEN 2 THEN 'leader'
        ELSE 'user'
    END,
    'AUDIT_EVENT_' || ((id % 20) + 1),
    'Automated batch audit event payload index ' || id
FROM generate_series(7, 10000) AS id;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
