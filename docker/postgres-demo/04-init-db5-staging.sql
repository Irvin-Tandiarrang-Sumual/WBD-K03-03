CREATE DATABASE db5_staging;
\c db5_staging;

CREATE TABLE staging_configs (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(80) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    environment VARCHAR(32) NOT NULL DEFAULT 'staging',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO staging_configs (key, value) VALUES
    ('app.debug', 'true'),
    ('feature.batch_v2', 'enabled'),
    ('feature.async_worker', 'active');

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
