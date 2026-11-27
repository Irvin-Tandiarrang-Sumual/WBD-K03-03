CREATE DATABASE db6_warehouse;
\c db6_warehouse;

CREATE TABLE raw_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'unprocessed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO raw_events (event_type, payload, status)
SELECT
    CASE (id % 4)
        WHEN 0 THEN 'user_signup'
        WHEN 1 THEN 'payment_processed'
        WHEN 2 THEN 'invoice_generated'
        ELSE 'page_view'
    END,
    json_build_object('event_id', id, 'client', 'web', 'score', id % 100)::jsonb,
    'unprocessed'
FROM generate_series(1, 2000) AS id;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
