package metrics

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	QueueJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "migration",
			Subsystem: "queue",
			Name:      "jobs",
			Help:      "Number of durable migration jobs by status.",
		},
		[]string{"status"},
	)

	QueueWaitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "migration",
			Subsystem: "queue",
			Name:      "wait_duration_seconds",
			Help:      "Time spent by a task waiting in the queue before execution.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	)

	LockWaitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "migration",
			Subsystem: "lock",
			Name:      "wait_duration_seconds",
			Help:      "Time spent waiting to acquire advisory lock by database dialect.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"dialect"},
	)

	ExecutionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "migration",
			Subsystem: "execution",
			Name:      "duration_seconds",
			Help:      "Duration of total migration execution by operation and status.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"operation", "status"},
	)

	BatchChunksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "migration",
			Subsystem: "batch",
			Name:      "chunks_total",
			Help:      "Total number of migration batch chunks processed by status.",
		},
		[]string{"status"},
	)

	BatchRowsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "migration",
			Subsystem: "batch",
			Name:      "rows_processed_total",
			Help:      "Total number of rows processed during batch migrations by operation.",
		},
		[]string{"operation"},
	)

	RetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "migration",
			Subsystem: "retry",
			Name:      "total",
			Help:      "Total number of retry attempts executed during migrations.",
		},
		[]string{"operation"},
	)

	DBPoolConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "db",
			Subsystem: "pool",
			Name:      "connections_active",
			Help:      "Current number of in-use active database connections.",
		},
		[]string{"dialect"},
	)

	DBPoolConnectionsIdle = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "db",
			Subsystem: "pool",
			Name:      "connections_idle",
			Help:      "Current number of idle database connections in the pool.",
		},
		[]string{"dialect"},
	)

	DBPoolConnectionsOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "db",
			Subsystem: "pool",
			Name:      "connections_open",
			Help:      "Current number of established open database connections.",
		},
		[]string{"dialect"},
	)

	DBPoolWaitDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "db",
			Subsystem: "pool",
			Name:      "wait_duration_seconds",
			Help:      "Cumulative connection wait time across currently open pools.",
		},
		[]string{"dialect"},
	)
)

var customRegistry = prometheus.NewRegistry()

func init() {
	RegisterMetrics(customRegistry)
}

func RegisterMetrics(r prometheus.Registerer) {
	r.MustRegister(
		QueueJobs,
		QueueWaitDuration,
		LockWaitDuration,
		ExecutionDuration,
		BatchChunksTotal,
		BatchRowsProcessedTotal,
		RetriesTotal,
		DBPoolConnectionsActive,
		DBPoolConnectionsIdle,
		DBPoolConnectionsOpen,
		DBPoolWaitDuration,
	)
}

func Registry() *prometheus.Registry {
	return customRegistry
}

type PrometheusMigrationMetrics struct{}

var _ ports.MigrationMetrics = (*PrometheusMigrationMetrics)(nil)

func NewPrometheusMigrationMetrics() *PrometheusMigrationMetrics {
	return &PrometheusMigrationMetrics{}
}

func (m *PrometheusMigrationMetrics) RecordExecution(operation, status string, durationSeconds float64) {
	if operation == "" {
		operation = ports.MetricOpExecute
	}
	ExecutionDuration.WithLabelValues(operation, status).Observe(durationSeconds)
}

func (m *PrometheusMigrationMetrics) RecordBatchChunk(status string, rowsAffected int64, operation string) {
	BatchChunksTotal.WithLabelValues(status).Inc()
	if rowsAffected > 0 {
		if operation == "" {
			operation = ports.MetricOpExecute
		}
		BatchRowsProcessedTotal.WithLabelValues(operation).Add(float64(rowsAffected))
	}
}

func (m *PrometheusMigrationMetrics) RecordRetry(operation string) {
	if operation == "" {
		operation = ports.MetricOpExecute
	}
	RetriesTotal.WithLabelValues(operation).Inc()
}
