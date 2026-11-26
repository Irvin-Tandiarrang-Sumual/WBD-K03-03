-- =======================================================
-- Database 3: Analytical Metrics & Historical Rollbacks
-- =======================================================
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

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;

-- =======================================================
-- Database 4: Billing System (Quarantine & Batch DML Demo)
-- =======================================================
CREATE DATABASE db4_billing;
\c db4_billing;

CREATE TABLE invoices (
    id BIGSERIAL PRIMARY KEY,
    invoice_number VARCHAR(64) NOT NULL UNIQUE,
    account_id BIGINT NOT NULL,
    amount NUMERIC(14, 2) NOT NULL,
    tax NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(32) NOT NULL DEFAULT 'unpaid',
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    invoice_id BIGINT NOT NULL REFERENCES invoices(id),
    payment_method VARCHAR(48) NOT NULL,
    paid_amount NUMERIC(14, 2) NOT NULL,
    reference_code VARCHAR(80) NOT NULL UNIQUE,
    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO invoices (invoice_number, account_id, amount, tax, status, due_date) VALUES
    ('INV-2026-001', 101, 1500000.00, 165000.00, 'paid', '2026-12-01'),
    ('INV-2026-002', 102, 3200000.00, 352000.00, 'paid', '2026-12-05'),
    ('INV-2026-003', 103, 850000.00, 93500.00, 'unpaid', '2026-12-10'),
    ('INV-2026-004', 104, 4500000.00, 495000.00, 'overdue', '2026-11-20'),
    ('INV-2026-005', 105, 1200000.00, 132000.00, 'paid', '2026-12-15'),
    ('INV-2026-006', 106, 2750000.00, 302500.00, 'unpaid', '2026-12-20'),
    ('INV-2026-007', 107, 6100000.00, 671000.00, 'paid', '2026-12-25'),
    ('INV-2026-008', 108, 980000.00, 107800.00, 'unpaid', '2026-12-28');

INSERT INTO payments (invoice_id, payment_method, paid_amount, reference_code) VALUES
    (1, 'bank_transfer', 1665000.00, 'TRX-BCA-8839201'),
    (2, 'credit_card', 3552000.00, 'TRX-VISA-4492104'),
    (5, 'e_wallet', 1332000.00, 'TRX-GOPAY-9921843'),
    (7, 'bank_transfer', 6771000.00, 'TRX-MANDIRI-2019482');

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
