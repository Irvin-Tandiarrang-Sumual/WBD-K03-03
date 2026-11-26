package ports

const (
	MetricStatusSuccess     = "success"
	MetricStatusFailed      = "failed"
	MetricStatusCommitted   = "committed"
	MetricStatusQuarantined = "quarantined"
	MetricStatusReverted    = "reverted"

	MetricOpExecute = "execute"
	MetricOpResume  = "resume"
	MetricOpRevert  = "revert"
)

type MigrationMetrics interface {
	RecordExecution(operation, status string, durationSeconds float64)
	RecordBatchChunk(status string, rowsAffected int64, operation string)
	RecordRetry(operation string)
}

type NoopMigrationMetrics struct{}

var _ MigrationMetrics = NoopMigrationMetrics{}

func (NoopMigrationMetrics) RecordExecution(string, string, float64) {}
func (NoopMigrationMetrics) RecordBatchChunk(string, int64, string)  {}
func (NoopMigrationMetrics) RecordRetry(string)                      {}
