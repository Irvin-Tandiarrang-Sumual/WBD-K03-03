package migration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	migrationpkg "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/service/migration"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestDiffSchemaSnapshots_BothNil_ReturnsIsIdenticalTrue(t *testing.T) {
	// Arrange
	var left *domain.SchemaSnapshot = nil
	var right *domain.SchemaSnapshot = nil

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if !diff.IsIdentical || len(diff.TableDiffs) != 0 {
		t.Fatalf("expected identical true for nil snapshots, got %+v", diff)
	}
}

func TestDiffSchemaSnapshots_IdenticalTables_ReturnsIsIdenticalTrue(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Schema: "public",
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
					{Name: "email", Type: "text", Nullable: true},
				},
				Indexes: []domain.SchemaIndexSnapshot{
					{Name: "users_pkey", Unique: true, Primary: true},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Schema: "public",
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
					{Name: "email", Type: "text", Nullable: true},
				},
				Indexes: []domain.SchemaIndexSnapshot{
					{Name: "users_pkey", Unique: true, Primary: true},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if !diff.IsIdentical || len(diff.TableDiffs) != 0 {
		t.Fatalf("expected identical true for matching snapshots, got %+v", diff)
	}
}

func TestDiffSchemaSnapshots_TableAdded_DetectsAddedTableAndCandidateCreateSQL(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"orders": {
				Schema: "public",
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "serial", Nullable: false},
					{Name: "total", Type: "numeric", Nullable: true},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || diff.TableDiffs[0].Status != "added" {
		t.Fatalf("expected table 'orders' to be marked as added, got %+v", diff)
	}
	if !strings.Contains(diff.CandidateSQL, "CREATE TABLE orders") {
		t.Fatalf("expected candidate SQL to contain 'CREATE TABLE orders', got %s", diff.CandidateSQL)
	}
}

func TestDiffSchemaSnapshots_TableRemoved_DetectsRemovedTableAndCandidateDropSQL(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"legacy_logs": {
				Schema: "public",
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || diff.TableDiffs[0].Status != "removed" {
		t.Fatalf("expected table 'legacy_logs' to be marked as removed, got %+v", diff)
	}
	if !strings.Contains(diff.CandidateSQL, "DROP TABLE legacy_logs;") {
		t.Fatalf("expected candidate SQL to contain 'DROP TABLE legacy_logs;', got %s", diff.CandidateSQL)
	}
}

func TestDiffSchemaSnapshots_ColumnAdded_DetectsAddedColumnAndCandidateAlterSQL(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
					{Name: "phone", Type: "varchar(20)", Nullable: true},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || len(diff.TableDiffs[0].AddedColumns) != 1 {
		t.Fatalf("expected column 'phone' to be added, got %+v", diff)
	}
	if !strings.Contains(diff.CandidateSQL, "ALTER TABLE users ADD COLUMN phone varchar(20) NULL;") {
		t.Fatalf("expected candidate SQL to add column 'phone', got %s", diff.CandidateSQL)
	}
}

func TestDiffSchemaSnapshots_ColumnRemoved_DetectsRemovedColumnAndCandidateAlterSQL(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
					{Name: "temp_token", Type: "text", Nullable: true},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "integer", Nullable: false},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || len(diff.TableDiffs[0].RemovedColumns) != 1 {
		t.Fatalf("expected column 'temp_token' to be removed, got %+v", diff)
	}
	if !strings.Contains(diff.CandidateSQL, "ALTER TABLE users DROP COLUMN temp_token;") {
		t.Fatalf("expected candidate SQL to drop column 'temp_token', got %s", diff.CandidateSQL)
	}
}

func TestDiffSchemaSnapshots_ColumnModified_DetectsModifiedTypeAndNullability(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"products": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "price", Type: "integer", Nullable: false},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"products": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "price", Type: "numeric(10,2)", Nullable: true},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || len(diff.TableDiffs[0].ModifiedColumns) != 1 {
		t.Fatalf("expected column 'price' to be modified, got %+v", diff)
	}
	mod := diff.TableDiffs[0].ModifiedColumns[0]
	if mod.Before.Type != "integer" || mod.After.Type != "numeric(10,2)" {
		t.Fatalf("expected modification from integer to numeric, got before: %s, after: %s", mod.Before.Type, mod.After.Type)
	}
}

func TestDiffSchemaSnapshots_IndexAdded_DetectsAddedIndex(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Indexes: []domain.SchemaIndexSnapshot{},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Indexes: []domain.SchemaIndexSnapshot{
					{Name: "idx_users_email", Unique: true, Primary: false},
				},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || len(diff.TableDiffs[0].AddedIndexes) != 1 {
		t.Fatalf("expected index 'idx_users_email' to be added, got %+v", diff)
	}
	if diff.TableDiffs[0].AddedIndexes[0].Name != "idx_users_email" {
		t.Fatalf("expected added index name 'idx_users_email', got %s", diff.TableDiffs[0].AddedIndexes[0].Name)
	}
}

func TestDiffSchemaSnapshots_IndexRemoved_DetectsRemovedIndex(t *testing.T) {
	// Arrange
	left := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Indexes: []domain.SchemaIndexSnapshot{
					{Name: "idx_users_old", Unique: false, Primary: false},
				},
			},
		},
	}
	right := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Indexes: []domain.SchemaIndexSnapshot{},
			},
		},
	}

	// Act
	diff := migrationpkg.DiffSchemaSnapshots(left, right)

	// Assert
	if diff.IsIdentical || len(diff.TableDiffs) != 1 || len(diff.TableDiffs[0].RemovedIndexes) != 1 {
		t.Fatalf("expected index 'idx_users_old' to be removed, got %+v", diff)
	}
	if diff.TableDiffs[0].RemovedIndexes[0].Name != "idx_users_old" {
		t.Fatalf("expected removed index name 'idx_users_old', got %s", diff.TableDiffs[0].RemovedIndexes[0].Name)
	}
}

func TestParseSchemaSnapshot_EmptyBytes_ReturnsNil(t *testing.T) {
	// Arrange
	emptyPayload := []byte{}

	// Act
	snapshot := migrationpkg.ParseSchemaSnapshot(emptyPayload)

	// Assert
	if snapshot != nil {
		t.Fatalf("expected nil for empty bytes, got %+v", snapshot)
	}
}

func TestParseSchemaSnapshot_InvalidJSON_ReturnsNil(t *testing.T) {
	// Arrange
	corruptPayload := []byte("{invalid-json")

	// Act
	snapshot := migrationpkg.ParseSchemaSnapshot(corruptPayload)

	// Assert
	if snapshot != nil {
		t.Fatalf("expected nil for invalid json, got %+v", snapshot)
	}
}

func TestToSnapshotJSON_NilSnapshot_ReturnsNil(t *testing.T) {
	// Arrange
	var nilSnapshot *domain.SchemaSnapshot = nil

	// Act
	bytes, err := migrationpkg.ToSnapshotJSON(nilSnapshot)

	// Assert
	if err != nil || bytes != nil {
		t.Fatalf("expected nil, nil for nil snapshot, got %v, %v", bytes, err)
	}
}

func TestToSnapshotJSON_ValidSnapshot_SerializesAndRoundtripsSuccessfully(t *testing.T) {
	// Arrange
	original := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"accounts": {
				Schema: "public",
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "bigint", Nullable: false},
					{Name: "balance", Type: "numeric", Nullable: true},
				},
				Indexes: []domain.SchemaIndexSnapshot{
					{Name: "accounts_pkey", Unique: true, Primary: true},
				},
			},
		},
	}

	// Act
	rawBytes, err := migrationpkg.ToSnapshotJSON(original)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}
	parsed := migrationpkg.ParseSchemaSnapshot(rawBytes)

	// Assert
	if parsed == nil || len(parsed.Tables) != 1 {
		t.Fatalf("expected 1 table after roundtrip, got %+v", parsed)
	}
	tbl, exists := parsed.Tables["accounts"]
	if !exists || len(tbl.Columns) != 2 || len(tbl.Indexes) != 1 {
		t.Fatalf("expected accounts table with 2 cols and 1 idx, got %+v", tbl)
	}
}

func TestFormatSnapshotAsBaselineSQL_EmptyOrNil_ReturnsDefaultComment(t *testing.T) {
	// Arrange
	var nilSnapshot *domain.SchemaSnapshot = nil

	// Act
	sql := migrationpkg.FormatSnapshotAsBaselineSQL(nilSnapshot)

	// Assert
	if !strings.Contains(sql, "Empty Database") {
		t.Fatalf("expected empty database comment, got '%s'", sql)
	}
}

func TestFormatSnapshotAsBaselineSQL_WithTables_GeneratesCreateTableStatements(t *testing.T) {
	// Arrange
	snapshot := &domain.SchemaSnapshot{
		Tables: map[string]domain.SchemaTableSnapshot{
			"users": {
				Columns: []domain.SchemaColumnSnapshot{
					{Name: "id", Type: "serial", Nullable: false},
					{Name: "bio", Type: "text", Nullable: true},
				},
			},
		},
	}

	// Act
	sql := migrationpkg.FormatSnapshotAsBaselineSQL(snapshot)

	// Assert
	if !strings.Contains(sql, "CREATE TABLE users") || !strings.Contains(sql, "id serial NOT NULL") || !strings.Contains(sql, "bio text NULL") {
		t.Fatalf("expected CREATE TABLE users with column definitions, got:\n%s", sql)
	}
}

func TestCaptureDatabaseSnapshot_NilInspector_ReturnsNil(t *testing.T) {
	// Arrange
	var nilInspector ports.SchemaInspectorRepository = nil

	// Act
	bytes, snapshot, err := migrationpkg.CaptureDatabaseSnapshot(context.Background(), nilInspector, domain.Database{})

	// Assert
	if err != nil || bytes != nil || snapshot != nil {
		t.Fatalf("expected nil, nil, nil for nil inspector, got bytes: %v, snap: %v, err: %v", bytes, snapshot, err)
	}
}

func TestCaptureDatabaseSnapshot_InspectorError_ReturnsError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("metadata query failed")
	driver := &testutil.DatabaseDriver{
		GetDatabaseMetadataFunc: func(_ context.Context, _ domain.Database) (*ports.DatabaseMetadata, error) {
			return nil, expectedErr
		},
	}

	// Act
	_, _, err := migrationpkg.CaptureDatabaseSnapshot(context.Background(), driver, domain.Database{})

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCaptureDatabaseSnapshot_ValidMetadata_BuildsSnapshot(t *testing.T) {
	// Arrange
	driver := &testutil.DatabaseDriver{
		GetDatabaseMetadataFunc: func(_ context.Context, _ domain.Database) (*ports.DatabaseMetadata, error) {
			return &ports.DatabaseMetadata{
				Tables: []ports.DatabaseTableMetadata{
					{Schema: "public", Name: "users"},
				},
				Columns: []ports.DatabaseColumnMetadata{
					{Table: "users", Name: "id", DataType: "integer", IsNullable: false},
				},
				Indexes: []ports.DatabaseIndexMetadata{
					{Table: "users", Name: "users_pkey", IsUnique: true, IsPrimary: true},
				},
			}, nil
		},
	}

	// Act
	bytes, snapshot, err := migrationpkg.CaptureDatabaseSnapshot(context.Background(), driver, domain.Database{})

	// Assert
	if err != nil || len(bytes) == 0 || snapshot == nil || len(snapshot.Tables) != 1 {
		t.Fatalf("expected valid captured snapshot, got err: %v, bytesLen: %d, snap: %+v", err, len(bytes), snapshot)
	}
}
