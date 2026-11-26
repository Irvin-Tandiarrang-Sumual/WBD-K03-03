package draft

import (
	"context"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func (s *draftService) prepareDraftSQL(ctx context.Context, input *domain.CreateDraftInput) error {
	validator, ok := s.databaseDriver.(ports.MigrationValidator)
	if !ok {
		return nil
	}
	db, err := s.databaseRepo.FindWithCredentialByID(ctx, input.DatabaseID)
	if err != nil {
		return err
	}
	if err := validateDraftSQL(ctx, validator, db, *input, input.SqlUp); err != nil {
		return err
	}
	s.resolveDraftReversibility(ctx, db, input)
	return validateDraftSQL(ctx, validator, db, *input, input.SqlDown)
}

func validateDraftSQL(ctx context.Context, validator ports.MigrationValidator, db domain.Database, input domain.CreateDraftInput, script string) error {
	if strings.TrimSpace(script) == "" {
		return nil
	}
	if input.IsBatch {
		batch, err := batchExecution(&domain.Draft{BatchColumn: input.BatchColumn, BatchSize: input.BatchSize}, script)
		if err != nil {
			return err
		}
		script = strings.TrimSuffix(batch.Statement, "IN ?") + "IN (0)"
	}
	return validator.ValidateMigrationSQL(ctx, db, script)
}

func (s *draftService) resolveDraftReversibility(ctx context.Context, db domain.Database, input *domain.CreateDraftInput) {
	reversible, reason := domain.ClassifyMigration(input.SqlUp)
	if reversible && input.IsReversible != nil && !*input.IsReversible {
		reversible = false
		reason = strings.TrimSpace(input.IrreversibleReason)
		if reason == "" {
			reason = "Marked irreversible by the author"
		}
	}
	if reversible && strings.TrimSpace(input.SqlDown) == "" {
		if !input.IsBatch {
			down, err := s.databaseDriver.GenerateSQLDown(ctx, db, input.SqlUp)
			if err == nil {
				input.SqlDown = down
			}
		}
		if strings.TrimSpace(input.SqlDown) == "" {
			reversible, reason = false, "No reversible SQL down is available"
		}
	}
	input.IsReversible = &reversible
	input.IrreversibleReason = reason
}
