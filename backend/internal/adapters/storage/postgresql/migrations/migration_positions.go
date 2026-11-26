package migrations

import (
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func backfillMigrationPositions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var databases []model.Database
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("current_migration_id IS NULL AND latest_execution_id=0 AND EXISTS (SELECT 1 FROM migrations WHERE migrations.database_id=databases.id AND status='success')").Find(&databases).Error; err != nil {
			return err
		}
		for _, database := range databases {
			var history []model.Migration
			if err := tx.Where("database_id=? AND status='success'", database.ID).Order("id").Find(&history).Error; err != nil {
				return err
			}
			var parent *uint
			for i := range history {
				m := &history[i]
				reversible, reason := domain.ClassifyMigration(m.SqlUp)
				if strings.HasPrefix(m.SqlDown, "-- Baseline schema snapshot") || strings.TrimSpace(m.SqlDown) == "" {
					reversible = false
					reason = "No inverse SQL available"
				}
				updates := map[string]any{"parent_migration_id": parent, "execution_id": m.ID, "is_reversible": reversible, "irreversible_reason": reason}
				if err := tx.Model(m).Updates(updates).Error; err != nil {
					return err
				}
				if parent != nil {
					if err := tx.Model(&model.Migration{}).Where("id=?", *parent).Update("forward_migration_id", m.ID).Error; err != nil {
						return err
					}
				}
				parent = &m.ID
			}
			if parent != nil {
				if err := tx.Model(&database).Updates(map[string]any{"current_migration_id": *parent, "latest_execution_id": *parent}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
