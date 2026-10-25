-- Case 1: db3_pg_empty intentionally has no user tables. It verifies that the
-- frontend handles a valid PostgreSQL connection with an empty schema.

-- Case 2: a second PostgreSQL database with related tables and seed data.
CREATE DATABASE db4_pg_sales OWNER demo_user;

\connect db4_pg_sales

CREATE TABLE sales_customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    region VARCHAR(40) NOT NULL
);

CREATE TABLE sales_orders (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES sales_customers(id),
    order_total NUMERIC(12, 2) NOT NULL,
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO sales_customers (name, region) VALUES
    ('Andi Wijaya', 'West'),
    ('Siti Rahma', 'East');

INSERT INTO sales_orders (customer_id, order_total) VALUES
    (1, 450000.00),
    (2, 725000.00);
