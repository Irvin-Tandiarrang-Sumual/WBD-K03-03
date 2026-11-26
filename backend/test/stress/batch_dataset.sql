\set ON_ERROR_STOP on
\timing on

CREATE TABLE wbd_batch_benchmark (
    id bigint PRIMARY KEY,
    score integer NOT NULL,
    payload text NOT NULL
);

INSERT INTO wbd_batch_benchmark
SELECT id, 0, repeat('x', 128)
FROM generate_series(1, :rows) AS id;

ANALYZE wbd_batch_benchmark;

BEGIN;
UPDATE wbd_batch_benchmark SET score = score + 1;
ROLLBACK;
