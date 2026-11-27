# Observability & Distributed Tracing Stack

Configuration and monitoring guide for the Wedebe platform using Prometheus, Grafana, Loki, Jaeger, and OpenTelemetry Collector.

---

## Starting the Observability Stack

Start the application services and observability infrastructure together using compose override files:

### Linux / macOS:
```bash
export OBSERVABILITY_CONFIG_DIR=./deploy/observability
docker compose -f docker-compose.yml -f deploy/observability/docker-compose.yaml -f deploy/observability/app.override.yaml up --build -d
```

### Windows (PowerShell):
```powershell
$env:OBSERVABILITY_CONFIG_DIR = './deploy/observability'
docker compose -f docker-compose.yml -f deploy/observability/docker-compose.yaml -f deploy/observability/app.override.yaml up --build -d
```

---

## Service Endpoints

Once all containers are running and healthy, access the monitoring services:

| Service | URL | Purpose |
|---|---|---|
| **Grafana** | [http://localhost:3001](http://localhost:3001) | Dashboards for migration throughput, worker queue depth, and latencies. |
| **Prometheus** | [http://localhost:9090](http://localhost:9090) | Metric scrape target for backend (`:8080/metrics`). |
| **Jaeger UI** | [http://localhost:16686](http://localhost:16686) | Distributed trace visualization for migration execution spans. |
| **Loki Logs** | via Grafana Explore | Centralized JSON application logs. |

---

## Structured Logging & Distributed Tracing

The backend writes structured JSON logs to `stdout` and copies them to a shared volume when `APP_LOG_PATH` is set. The OpenTelemetry Collector forwards log records to Loki and application traces to Jaeger.

### Querying Logs in Loki
In Grafana Explore, query application logs with:
```text
{service_name="wbd-backend"} | json
```

Example batch chunk log output:
```json
{
  "event": "sql.batch.chunk.complete",
  "status": "success",
  "correlation_id": "wbd-demo-logging-1795736006689804900-execute",
  "trace_id": "2d871c5c8a4bf49c87f479caa88d2dcf",
  "span_id": "5deeb60f818c6532",
  "job_id": 1,
  "batch_number": 3,
  "attempt": 1,
  "rows": 6,
  "query": "UPDATE [redacted]"
}
```

SQL audit records record operational metadata while redacting statement contents and sensitive parameter values.

### Distributed Traces in Jaeger
A single trace ID covers the entire job processing lifecycle:
- `http.execute_request`: Initial client HTTP request.
- `queue.submit`: Job creation in the migration queue.
- `queue.lock`: PostgreSQL advisory lock acquisition.
- `worker.execute_task`: Task pickup by background worker thread.
- `db.batch.prepare`: Batch cursor initialization and chunk segmentation.
- `db.batch.chunk`: Individual batch chunk transaction commit.

### Core Prometheus Metrics
- `migration_queue_jobs{status="done"}`: Total completed migration jobs.
- `migration_batch_rows_processed_total{operation="execute"}`: Total rows processed by batch DML operations.
- `db_pool_connections_idle{dialect="postgresql"}`: Connection pool idle vs active counts.
