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

INSERT INTO invoices (invoice_number, account_id, amount, tax, status, due_date)
SELECT
    'INV-2026-' || LPAD(id::text, 8, '0'),
    (100 + (id % 500)),
    ((id * 31 % 10000000) + 100000)::numeric(14,2),
    ((id * 31 % 10000000) * 0.11)::numeric(10,2),
    CASE (id % 3)
        WHEN 0 THEN 'paid'
        WHEN 1 THEN 'unpaid'
        ELSE 'overdue'
    END,
    (CURRENT_DATE - ((id % 60) || ' days')::interval)::date
FROM generate_series(9, 100000) AS id;

INSERT INTO payments (invoice_id, payment_method, paid_amount, reference_code)
SELECT
    id,
    CASE (id % 3)
        WHEN 0 THEN 'bank_transfer'
        WHEN 1 THEN 'credit_card'
        ELSE 'e_wallet'
    END,
    ((id * 31 % 10000000) + 100000)::numeric(14,2),
    'TRX-AUTO-' || LPAD(id::text, 8, '0')
FROM generate_series(8, 100000) AS id;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
