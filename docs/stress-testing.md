# Performance & Stress Testing Guide (k6)

Load and stress testing suite designed to evaluate system stability, concurrent API traffic handling, PostgreSQL advisory lock contention, worker queue capacity, and Mean Time to Recovery (MTTR) under high concurrency.

---

## 1. How to Run Tests

### Option A: Docker Container Mode (Zero-Install)
Runs the test suite inside the official k6 container (`grafana/k6:latest`) configured with high file descriptor limits (`nofile: 65535`):

```bash
# Run master stress suite in Docker:
make test-stress-docker

# Run with clean database volumes:
make test-stress-fresh
```

---

### Option B: Local CLI Mode
If `k6` is installed locally:

```bash
# Run master suite (3 scenarios concurrently):
k6 run -e BASE_URL=http://localhost:8080 stress/stress_suite.js

# Run individual scenarios:
k6 run -e BASE_URL=http://localhost:8080 stress/scenarios/api_traffic.js
k6 run -e BASE_URL=http://localhost:8080 stress/scenarios/same_db_contention.js
k6 run -e BASE_URL=http://localhost:8080 stress/scenarios/cross_db_saturation.js
```

---

## 2. Load Architecture & Staged Profile

The suite applies load across 6 stages (ramp-up, peak saturation soak, and ramp-down) up to a maximum of **75 Virtual Users (VUs)**:

```text
VUs / RPS
   ▲
75 ┼                       ┌──────────────┐ (Stage 4: Peak Saturation Soak)
   │                      ╱                ╲
30 ┼         ┌───────────┘                  ╲
   │        ╱ (Stage 2: Above Normal)        ╲
10 ┼───────┘                                  └───────┐ (Stage 5: Ramp Down)
   │ (Stage 1: Normal Baseline)                       └────────► Time
 0 ┴─────┬───────────┬───────────┬──────────┬─────────┬─────────►
        30s         1m30s       2m30s      3m00s     3m30s     4m15s
     [Normal]   [Above Normal]  [Stress]   [Soak]  [Recovery] [MTTR Probe]
```

### Scenario Breakdown
1. **`scenarios/api_traffic.js`**:
   - Simulates interactive user operations: database listing, schema inspection, migration history queries, and analytical queries using `EXPLAIN (ANALYZE, BUFFERS)`.
2. **`scenarios/same_db_contention.js`**:
   - Executes concurrent migration drafts targeting the **same database** (`db1_pg`).
   - Verifies that PostgreSQL advisory locks (`pg_try_advisory_lock`) reject concurrent executions with `HTTP 409 Conflict` without deadlocking.
3. **`scenarios/cross_db_saturation.js`**:
   - Submits migration jobs across **multiple distinct databases** (`db1_pg`, `db3_analytics`, `db4_billing`).
   - Tests background worker pool concurrency (2 worker threads) and queue behavior.
4. **`helpers/job_poller.js`**:
   - Performs non-blocking polling until each job reaches its true terminal state (`applied`, `failed`, or `quarantined`). Initial `HTTP 200` or `202` responses are not treated as execution success.

---

## 3. Benchmark Results and SLO Evaluation

Empirical benchmark measurements from load testing runs:

| Evaluation Metric | SLO Target | Observed Result (Peak 75 VUs) | Status | Notes |
|---|---|---|---|---|
| **API Response Time (p95)** | < 500 ms | **41.20 ms - 43.72 ms** | PASSED | API latencies stayed under 45 ms across all phases. |
| **Unexpected Errors** | < 1% | **0.00%** | PASSED | Zero unexpected 5xx responses. |
| **Lock Contention Handling** | 100% Non-Deadlock | **100% Handled (409 Conflict)** | PASSED | Conflicting migrations on the same target rejected cleanly. |
| **Queue Worker Throughput** | > 2.5 jobs/sec | **2.77 - 4.97 jobs/sec** | PASSED | Workers processed jobs continuously without starvation. |
| **Mean Time to Recovery (MTTR)**| < 10 seconds | **< 3.2 seconds** | PASSED | System returned to baseline latencies immediately after peak load. |
