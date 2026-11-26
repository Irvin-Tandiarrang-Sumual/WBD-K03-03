package postgresql

import (
	"context"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"gorm.io/gorm"
)

func CollectMetrics(ctx context.Context, db *gorm.DB) error {
	var counts []struct {
		Status string
		Count  int64
	}
	err := db.WithContext(ctx).Model(&model.MigrationJob{}).
		Select("status, count(*) AS count").Group("status").Scan(&counts).Error
	if err != nil {
		return err
	}
	totals := make(map[string]int64, len(counts))
	for _, count := range counts {
		totals[count.Status] = count.Count
	}
	metrics.QueueJobs.WithLabelValues("waiting").Set(float64(totals[jobPending]))
	for _, status := range []string{jobRunning, jobDone, jobFailed, jobUncertain} {
		metrics.QueueJobs.WithLabelValues(status).Set(float64(totals[status]))
	}
	return nil
}
