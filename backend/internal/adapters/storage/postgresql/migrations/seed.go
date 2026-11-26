package migrations

import (
	"errors"
	"fmt"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/hasher"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedTestData(db *gorm.DB) error {
	return seedData(db, func(password string) (string, error) {
		return password, nil
	})
}

func SeedDevelopmentData(db *gorm.DB, cipher ports.DatabaseCredentialCipher) error {
	return seedData(db, cipher.Encrypt)
}

func seedData(db *gorm.DB, encodePassword func(string) (string, error)) error {
	return db.Transaction(func(tx *gorm.DB) error {
		databases, err := seedTestDatabases(tx, encodePassword)
		if err != nil {
			return err
		}

		users, err := seedUsers(tx)
		if err != nil {
			return err
		}

		groups, err := seedGroups(tx, databases, users)
		if err != nil {
			return err
		}

		if err := associateUsersAndGroups(tx, users, groups); err != nil {
			return err
		}

		migrations, err := seedMigrationHistory(tx, databases, users)
		if err != nil {
			return err
		}

		if err := seedDraftsAndApprovals(tx, databases, users, groups, migrations); err != nil {
			return err
		}

		return nil
	})
}

func seedTestDatabases(tx *gorm.DB, encodePassword func(string) (string, error)) (map[string]model.Database, error) {
	databaseSeeds := []model.Database{
		{Port: 5432, Host: "demo_postgres", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db1_pg"},
		{Port: 3306, Host: "demo_mysql", User: "demo_user", Password: "demo_password", Type: "mysql", Name: "db2_mysql"},
		{Port: 5432, Host: "demo_postgres", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db3_analytics"},
		{Port: 5432, Host: "demo_postgres", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db4_billing"},
	}

	databases := make(map[string]model.Database, len(databaseSeeds))
	for _, databaseSeed := range databaseSeeds {
		encodedPassword, encodeErr := encodePassword(databaseSeed.Password)
		if encodeErr != nil {
			return nil, fmt.Errorf("encoding password for database %s: %w", databaseSeed.Name, encodeErr)
		}
		databaseSeed.Password = encodedPassword

		var database model.Database
		err := tx.Where("name = ?", databaseSeed.Name).First(&database).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			database = databaseSeed
			err = tx.Create(&database).Error
		} else if err == nil {
			err = tx.Model(&database).Updates(map[string]any{
				"host":     databaseSeed.Host,
				"user":     databaseSeed.User,
				"password": databaseSeed.Password,
				"type":     databaseSeed.Type,
				"port":     databaseSeed.Port,
			}).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding database %s: %w", databaseSeed.Name, err)
		}
		databases[databaseKey(databaseSeed.Name)] = database
	}
	return databases, nil
}

func seedUsers(tx *gorm.DB) (map[string]model.User, error) {
	userSeeds := []struct {
		username string
		password string
		role     string
	}{
		{"admin", "admin123", "admin"},
		{"leader", "leader123", "user"},
		{"user", "user123", "user"},
		{"leader_qa", "qa123", "user"},
		{"auditor_lead", "auditor123", "user"},
		{"auditor", "auditor123", "user"},
	}

	users := make(map[string]model.User, len(userSeeds))
	for _, seed := range userSeeds {
		var user model.User
		err := tx.Where("username = ?", seed.username).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hashed, err := hasher.HashPassword(seed.password)
			if err != nil {
				return nil, fmt.Errorf("hashing password for %s: %w", seed.username, err)
			}
			user = model.User{
				Username: seed.username,
				Password: hashed,
				Role:     seed.role,
			}
			if err := tx.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error; err != nil {
				return nil, fmt.Errorf("seeding user %s: %w", seed.username, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("finding user %s: %w", seed.username, err)
		}
		users[seed.username] = user
	}
	return users, nil
}

func seedGroups(tx *gorm.DB, databases map[string]model.Database, users map[string]model.User) (map[string]model.Group, error) {
	adminUser := users["admin"]
	leaderUser := users["leader"]
	leaderQA := users["leader_qa"]
	auditorLead := users["auditor_lead"]

	db1 := databases[databaseKey("db1_pg")]
	db2 := databases[databaseKey("db2_mysql")]
	db3 := databases[databaseKey("db3_analytics")]
	db4 := databases[databaseKey("db4_billing")]

	type dbAccess struct {
		db         model.Database
		accessType string
	}

	groupSeeds := []struct {
		key      string
		name     string
		leaderID uint
		accesses []dbAccess
	}{
		{
			key:      "admin",
			name:     "admin",
			leaderID: adminUser.ID,
			accesses: []dbAccess{
				{db: db1, accessType: "view_edit"},
				{db: db2, accessType: "view_edit"},
				{db: db3, accessType: "view_edit"},
				{db: db4, accessType: "view_edit"},
			},
		},
		{
			key:      "engineering",
			name:     "engineering",
			leaderID: leaderUser.ID,
			accesses: []dbAccess{
				{db: db1, accessType: "view_edit"},
				{db: db2, accessType: "view_edit"},
				{db: db3, accessType: "view_only"},
				{db: db4, accessType: "view_edit"},
			},
		},
		{
			key:      "qa-team",
			name:     "qa-team",
			leaderID: leaderQA.ID,
			accesses: []dbAccess{
				{db: db1, accessType: "view_edit"},
				{db: db3, accessType: "view_only"},
				{db: db4, accessType: "view_edit"},
			},
		},
		{
			key:      "auditors",
			name:     "auditors",
			leaderID: auditorLead.ID,
			accesses: []dbAccess{
				{db: db1, accessType: "view_only"},
				{db: db2, accessType: "view_only"},
				{db: db3, accessType: "view_only"},
				{db: db4, accessType: "view_only"},
			},
		},
	}

	groups := make(map[string]model.Group, len(groupSeeds))
	for _, groupSeed := range groupSeeds {
		var group model.Group
		err := tx.Where("name = ?", groupSeed.name).First(&group).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			group = model.Group{Name: groupSeed.name, LeaderID: groupSeed.leaderID}
			err = tx.Create(&group).Error
		} else if err == nil {
			err = tx.Model(&group).Update("leader_id", groupSeed.leaderID).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding %s group: %w", groupSeed.key, err)
		}

		if err := tx.Where("group_id = ?", group.ID).Delete(&model.GroupDatabase{}).Error; err != nil {
			return nil, fmt.Errorf("clearing %s group databases: %w", groupSeed.key, err)
		}
		for _, acc := range groupSeed.accesses {
			if err := tx.Create(&model.GroupDatabase{
				GroupID: group.ID, DatabaseID: acc.db.ID, AccessType: acc.accessType,
			}).Error; err != nil {
				return nil, fmt.Errorf("seeding %s group database %s: %w", groupSeed.key, acc.db.Name, err)
			}
		}
		groups[groupSeed.key] = group
	}
	return groups, nil
}

func associateUsersAndGroups(tx *gorm.DB, users map[string]model.User, groups map[string]model.Group) error {
	associations := []struct {
		user  model.User
		group model.Group
	}{
		{users["admin"], groups["admin"]},
		{users["leader"], groups["engineering"]},
		{users["user"], groups["engineering"]},
		{users["leader_qa"], groups["qa-team"]},
		{users["auditor_lead"], groups["auditors"]},
		{users["auditor"], groups["auditors"]},
	}

	for _, assoc := range associations {
		if err := tx.Model(&assoc.user).Association("Groups").Replace([]model.Group{{ID: assoc.group.ID}}); err != nil {
			return fmt.Errorf("associating user %s to group %s: %w", assoc.user.Username, assoc.group.Name, err)
		}
	}
	return nil
}

func seedMigrationHistory(
	tx *gorm.DB,
	databases map[string]model.Database,
	users map[string]model.User,
) (map[string]model.Migration, error) {
	adminUser := users["admin"]
	userUser := users["user"]

	db1 := databases[databaseKey("db1_pg")]
	db2 := databases[databaseKey("db2_mysql")]
	db3 := databases[databaseKey("db3_analytics")]
	db4 := databases[databaseKey("db4_billing")]

	now := time.Now().UTC()
	migrations := make(map[string]model.Migration)

	// --- DB1 (PostgreSQL): 3 Migrations (All Reversible, Head at v3) ---
	m1 := model.Migration{
		DatabaseID:   int(db1.ID),
		Version:      "1",
		ExecutionID:  1,
		Status:       "success",
		SqlUp:        "CREATE TABLE customers (\n    id BIGSERIAL PRIMARY KEY,\n    name VARCHAR(120) NOT NULL,\n    email VARCHAR(255) NOT NULL UNIQUE,\n    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);",
		SqlDown:      "DROP TABLE customers;",
		IsReversible: true,
		UserID:       int(adminUser.ID),
		MigratedAt:   now.Add(-6 * time.Hour),
		Snapshot:     datatypes.JSON([]byte(`{"tables": ["customers"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m1.DatabaseID, m1.Version, m1.ExecutionID).FirstOrCreate(&m1).Error; err != nil {
		return nil, err
	}
	migrations["db1_v1"] = m1

	m2 := model.Migration{
		DatabaseID:        int(db1.ID),
		Version:           "2",
		ExecutionID:       2,
		ParentMigrationID: &m1.ID,
		Status:            "success",
		SqlUp:             "CREATE TABLE orders (\n    id BIGSERIAL PRIMARY KEY,\n    customer_id BIGINT NOT NULL REFERENCES customers(id),\n    total_amount NUMERIC(12, 2) NOT NULL,\n    status VARCHAR(32) NOT NULL DEFAULT 'pending',\n    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);",
		SqlDown:           "DROP TABLE orders;",
		IsReversible:      true,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-4 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["customers", "orders"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m2.DatabaseID, m2.Version, m2.ExecutionID).FirstOrCreate(&m2).Error; err != nil {
		return nil, err
	}
	migrations["db1_v2"] = m2

	m3 := model.Migration{
		DatabaseID:        int(db1.ID),
		Version:           "3",
		ExecutionID:       3,
		ParentMigrationID: &m2.ID,
		Status:            "success",
		SqlUp:             "CREATE TABLE order_items (\n    id BIGSERIAL PRIMARY KEY,\n    order_id BIGINT NOT NULL REFERENCES orders(id),\n    item_name VARCHAR(120) NOT NULL,\n    quantity INT NOT NULL DEFAULT 1,\n    price NUMERIC(12, 2) NOT NULL\n);",
		SqlDown:           "DROP TABLE order_items;",
		IsReversible:      true,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-2 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["customers", "orders", "order_items"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m3.DatabaseID, m3.Version, m3.ExecutionID).FirstOrCreate(&m3).Error; err != nil {
		return nil, err
	}
	migrations["db1_v3"] = m3

	// Update DB1 HEAD
	if err := tx.Model(&model.Database{}).Where("id = ?", db1.ID).Updates(map[string]any{
		"current_migration_id": m3.ID,
		"latest_execution_id":  3,
	}).Error; err != nil {
		return nil, err
	}

	// --- DB2 (MySQL): 3 Migrations (v3 is Irreversible!) ---
	m4 := model.Migration{
		DatabaseID:   int(db2.ID),
		Version:      "1",
		ExecutionID:  1,
		Status:       "success",
		SqlUp:        "CREATE TABLE products (\n    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,\n    sku VARCHAR(40) NOT NULL UNIQUE,\n    name VARCHAR(120) NOT NULL,\n    price DECIMAL(12, 2) NOT NULL,\n    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP\n);",
		SqlDown:      "DROP TABLE products;",
		IsReversible: true,
		UserID:       int(adminUser.ID),
		MigratedAt:   now.Add(-6 * time.Hour),
		Snapshot:     datatypes.JSON([]byte(`{"tables": ["products"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m4.DatabaseID, m4.Version, m4.ExecutionID).FirstOrCreate(&m4).Error; err != nil {
		return nil, err
	}
	migrations["db2_v1"] = m4

	m5 := model.Migration{
		DatabaseID:        int(db2.ID),
		Version:           "2",
		ExecutionID:       2,
		ParentMigrationID: &m4.ID,
		Status:            "success",
		SqlUp:             "CREATE TABLE inventory_movements (\n    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,\n    product_id BIGINT UNSIGNED NOT NULL,\n    quantity_change INT NOT NULL,\n    movement_type VARCHAR(24) NOT NULL,\n    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\n    CONSTRAINT inventory_product_fk FOREIGN KEY (product_id) REFERENCES products(id)\n);",
		SqlDown:           "DROP TABLE inventory_movements;",
		IsReversible:      true,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-3 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["products", "inventory_movements"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m5.DatabaseID, m5.Version, m5.ExecutionID).FirstOrCreate(&m5).Error; err != nil {
		return nil, err
	}
	migrations["db2_v2"] = m5

	m6 := model.Migration{
		DatabaseID:         int(db2.ID),
		Version:            "3",
		ExecutionID:        3,
		ParentMigrationID:  &m5.ID,
		Status:             "success",
		SqlUp:              "ALTER TABLE products DROP COLUMN legacy_sku;",
		SqlDown:            "",
		IsReversible:       false,
		IrreversibleReason: "Cannot restore dropped column legacy_sku without data loss",
		UserID:             int(adminUser.ID),
		MigratedAt:         now.Add(-1 * time.Hour),
		Snapshot:           datatypes.JSON([]byte(`{"tables": ["products", "inventory_movements"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m6.DatabaseID, m6.Version, m6.ExecutionID).FirstOrCreate(&m6).Error; err != nil {
		return nil, err
	}
	migrations["db2_v3"] = m6

	// Update DB2 HEAD
	if err := tx.Model(&model.Database{}).Where("id = ?", db2.ID).Updates(map[string]any{
		"current_migration_id": m6.ID,
		"latest_execution_id":  3,
	}).Error; err != nil {
		return nil, err
	}

	// --- DB3 (PostgreSQL Analytics): Post-Rollback State with Active Forward Redo Pointer! ---
	m7 := model.Migration{
		DatabaseID:   int(db3.ID),
		Version:      "1",
		ExecutionID:  1,
		Status:       "success",
		SqlUp:        "CREATE TABLE daily_metrics (\n    id BIGSERIAL PRIMARY KEY,\n    metric_date DATE NOT NULL,\n    active_users INT NOT NULL DEFAULT 0,\n    total_sales NUMERIC(14, 2) NOT NULL DEFAULT 0.00,\n    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);\nCREATE TABLE audit_logs (\n    id BIGSERIAL PRIMARY KEY,\n    actor VARCHAR(80) NOT NULL,\n    action VARCHAR(120) NOT NULL,\n    details TEXT,\n    logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);",
		SqlDown:      "DROP TABLE daily_metrics;\nDROP TABLE audit_logs;",
		IsReversible: true,
		UserID:       int(adminUser.ID),
		MigratedAt:   now.Add(-6 * time.Hour),
		Snapshot:     datatypes.JSON([]byte(`{"tables": ["daily_metrics", "audit_logs"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m7.DatabaseID, m7.Version, m7.ExecutionID).FirstOrCreate(&m7).Error; err != nil {
		return nil, err
	}
	migrations["db3_v1"] = m7

	m8 := model.Migration{
		DatabaseID:        int(db3.ID),
		Version:           "2",
		ExecutionID:       2,
		ParentMigrationID: &m7.ID,
		Status:            "success",
		SqlUp:             "CREATE TABLE temp_import_staging (\n    raw_id INT,\n    payload TEXT\n);",
		SqlDown:           "DROP TABLE temp_import_staging;",
		IsReversible:      true,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-4 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["daily_metrics", "audit_logs", "temp_import_staging"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m8.DatabaseID, m8.Version, m8.ExecutionID).FirstOrCreate(&m8).Error; err != nil {
		return nil, err
	}
	migrations["db3_v2"] = m8

	m9Rollback := model.Migration{
		DatabaseID:         int(db3.ID),
		Version:            "1",
		ExecutionID:        3,
		ParentMigrationID:  &m7.ID,
		RevertsMigrationID: &m8.ID,
		ForwardMigrationID: &m8.ID, // Enables Forward (Redo) from rollback position!
		Status:             "success",
		SqlUp:              "DROP TABLE temp_import_staging;",
		SqlDown:            "CREATE TABLE temp_import_staging (\n    raw_id INT,\n    payload TEXT\n);",
		IsReversible:       true,
		UserID:             int(adminUser.ID),
		MigratedAt:         now.Add(-2 * time.Hour),
		Snapshot:           datatypes.JSON([]byte(`{"tables": ["daily_metrics", "audit_logs"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m9Rollback.DatabaseID, m9Rollback.Version, m9Rollback.ExecutionID).FirstOrCreate(&m9Rollback).Error; err != nil {
		return nil, err
	}
	migrations["db3_v1_rollback"] = m9Rollback

	// Update DB3 HEAD to point to m9Rollback (so Forward (Redo) is active)
	if err := tx.Model(&model.Database{}).Where("id = ?", db3.ID).Updates(map[string]any{
		"current_migration_id": m9Rollback.ID,
		"latest_execution_id":  3,
	}).Error; err != nil {
		return nil, err
	}

	// --- DB4 (PostgreSQL Billing): Includes a Failed Migration Record ---
	m10 := model.Migration{
		DatabaseID:   int(db4.ID),
		Version:      "1",
		ExecutionID:  1,
		Status:       "success",
		SqlUp:        "CREATE TABLE invoices (\n    id BIGSERIAL PRIMARY KEY,\n    invoice_number VARCHAR(64) NOT NULL UNIQUE,\n    account_id BIGINT NOT NULL,\n    amount NUMERIC(14, 2) NOT NULL,\n    tax NUMERIC(10, 2) NOT NULL DEFAULT 0.00,\n    status VARCHAR(32) NOT NULL DEFAULT 'unpaid',\n    due_date DATE NOT NULL,\n    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);",
		SqlDown:      "DROP TABLE invoices;",
		IsReversible: true,
		UserID:       int(adminUser.ID),
		MigratedAt:   now.Add(-5 * time.Hour),
		Snapshot:     datatypes.JSON([]byte(`{"tables": ["invoices"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m10.DatabaseID, m10.Version, m10.ExecutionID).FirstOrCreate(&m10).Error; err != nil {
		return nil, err
	}
	migrations["db4_v1"] = m10

	m11 := model.Migration{
		DatabaseID:        int(db4.ID),
		Version:           "2",
		ExecutionID:       2,
		ParentMigrationID: &m10.ID,
		Status:            "success",
		SqlUp:             "CREATE TABLE payments (\n    id BIGSERIAL PRIMARY KEY,\n    invoice_id BIGINT NOT NULL REFERENCES invoices(id),\n    payment_method VARCHAR(48) NOT NULL,\n    paid_amount NUMERIC(14, 2) NOT NULL,\n    reference_code VARCHAR(80) NOT NULL UNIQUE,\n    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n);",
		SqlDown:           "DROP TABLE payments;",
		IsReversible:      true,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-3 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["invoices", "payments"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m11.DatabaseID, m11.Version, m11.ExecutionID).FirstOrCreate(&m11).Error; err != nil {
		return nil, err
	}
	migrations["db4_v2"] = m11

	// Failed migration record (Tests red 'Fail' badge and error message inspection)
	m12Fail := model.Migration{
		DatabaseID:        int(db4.ID),
		Version:           "3",
		ExecutionID:       3,
		ParentMigrationID: &m11.ID,
		Status:            "fail",
		SqlUp:             "ALTER TABLE invoices ADD COLUMN tax_code VARCHAR(32) REFERENCES non_existent_taxes(id);",
		SqlDown:           "",
		ErrorMessage:      "pq: relation \"non_existent_taxes\" does not exist",
		IsReversible:      false,
		UserID:            int(userUser.ID),
		MigratedAt:        now.Add(-1 * time.Hour),
		Snapshot:          datatypes.JSON([]byte(`{"tables": ["invoices", "payments"]}`)),
	}
	if err := tx.Where("database_id = ? AND version = ? AND execution_id = ?", m12Fail.DatabaseID, m12Fail.Version, m12Fail.ExecutionID).FirstOrCreate(&m12Fail).Error; err != nil {
		return nil, err
	}
	migrations["db4_v3_fail"] = m12Fail

	// Update DB4 HEAD to point to last successful migration (m11)
	if err := tx.Model(&model.Database{}).Where("id = ?", db4.ID).Updates(map[string]any{
		"current_migration_id": m11.ID,
		"latest_execution_id":  3,
	}).Error; err != nil {
		return nil, err
	}

	return migrations, nil
}

func seedDraftsAndApprovals(
	tx *gorm.DB,
	databases map[string]model.Database,
	users map[string]model.User,
	groups map[string]model.Group,
	migrations map[string]model.Migration,
) error {
	userUser := users["user"]
	leaderUser := users["leader"]
	leaderQA := users["leader_qa"]

	db1 := databases[databaseKey("db1_pg")]
	db2 := databases[databaseKey("db2_mysql")]
	db4 := databases[databaseKey("db4_billing")]

	grpEng := groups["engineering"]
	grpQA := groups["qa-team"]

	now := time.Now().UTC()
	respTime := now.Add(-15 * time.Minute)

	// --- DRAFT 1: Multi-group consensus pending approval on db1_pg ---
	// Partial consensus state: Tim Engineering APPROVED, Tim QA is still PENDING!
	d1 := model.Draft{
		DatabaseID:         db1.ID,
		UserID:             userUser.ID,
		SqlUp:              "ALTER TABLE customers ADD COLUMN phone_number VARCHAR(32) NULL;\nCREATE INDEX idx_customers_phone ON customers(phone_number);",
		SqlDown:            "ALTER TABLE customers DROP COLUMN phone_number;",
		IsReversible:       true,
		Status:             "pending_approval",
		CurrentRound:       1,
		CreatedAt:          now.Add(-45 * time.Minute),
		UpdatedAt:          now.Add(-45 * time.Minute),
	}
	if err := tx.Where("database_id = ? AND sql_up LIKE ?", d1.DatabaseID, "ALTER TABLE customers ADD COLUMN phone_number%").FirstOrCreate(&d1).Error; err != nil {
		return err
	}

	// Leader Engineering already voted APPROVED:
	_ = tx.Where("draft_id = ? AND group_id = ?", d1.ID, grpEng.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d1.ID,
		GroupID:     grpEng.ID,
		Round:       1,
		LeaderID:    &leaderUser.ID,
		Phase:       "initial",
		Status:      "approved",
		Vote:        "approved",
		RespondedAt: &respTime,
	})
	// Leader QA is still PENDING:
	_ = tx.Where("draft_id = ? AND group_id = ?", d1.ID, grpQA.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:  d1.ID,
		GroupID:  grpQA.ID,
		Round:    1,
		LeaderID: &leaderQA.ID,
		Phase:    "initial",
		Status:   "pending",
		Vote:     "pending",
	})

	// --- DRAFT 2: Approved DDL ready for execution on db1_pg ---
	d2 := model.Draft{
		DatabaseID:         db1.ID,
		UserID:             userUser.ID,
		SqlUp:              "CREATE INDEX idx_orders_customer_id ON orders(customer_id);",
		SqlDown:            "DROP INDEX idx_orders_customer_id;",
		IsReversible:       true,
		Status:             "approved",
		CurrentRound:       1,
		CreatedAt:          now.Add(-90 * time.Minute),
		UpdatedAt:          now.Add(-15 * time.Minute),
	}
	if err := tx.Where("database_id = ? AND sql_up LIKE ?", d2.DatabaseID, "CREATE INDEX idx_orders_customer_id%").FirstOrCreate(&d2).Error; err != nil {
		return err
	}

	_ = tx.Where("draft_id = ? AND group_id = ?", d2.ID, grpEng.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d2.ID,
		GroupID:     grpEng.ID,
		Round:       1,
		LeaderID:    &leaderUser.ID,
		Phase:       "initial",
		Status:      "approved",
		Vote:        "approved",
		RespondedAt: &respTime,
	})
	_ = tx.Where("draft_id = ? AND group_id = ?", d2.ID, grpQA.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d2.ID,
		GroupID:     grpQA.ID,
		Round:       1,
		LeaderID:    &leaderQA.ID,
		Phase:       "initial",
		Status:      "approved",
		Vote:        "approved",
		RespondedAt: &respTime,
	})

	// --- DRAFT 3: Historical applied draft on db1_pg (Tests 'applied' filter) ---
	m3 := migrations["db1_v3"]
	d3Applied := model.Draft{
		DatabaseID:         db1.ID,
		UserID:             userUser.ID,
		SqlUp:              m3.SqlUp,
		SqlDown:            m3.SqlDown,
		IsReversible:       true,
		Status:             "applied",
		CurrentRound:       1,
		CreatedAt:          now.Add(-3 * time.Hour),
		UpdatedAt:          now.Add(-2 * time.Hour),
	}
	_ = tx.Where("database_id = ? AND status = 'applied'", db1.ID).FirstOrCreate(&d3Applied)

	// --- DRAFT 4: Batch DML in partial execution state on db4_billing ---
	// Triggers Quarantine on db4_billing!
	d4Batch := model.Draft{
		DatabaseID:         db4.ID,
		UserID:             userUser.ID,
		SqlUp:              "-- Process monthly recurring billing accounts\nUPDATE invoices SET status = 'overdue' WHERE due_date < NOW() AND status = 'unpaid' AND id BETWEEN ? AND ?;",
		IsBatch:            true,
		BatchColumn:        "id",
		BatchSize:          500,
		Status:             "partial",
		RecoveryAction:     "resume",
		CurrentRound:       1,
		CreatedAt:          now.Add(-2 * time.Hour),
		UpdatedAt:          now.Add(-10 * time.Minute),
	}
	if err := tx.Where("database_id = ? AND sql_up LIKE ?", d4Batch.DatabaseID, "%Process monthly recurring billing%").FirstOrCreate(&d4Batch).Error; err != nil {
		return err
	}

	_ = tx.Where("draft_id = ? AND group_id = ?", d4Batch.ID, grpEng.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d4Batch.ID,
		GroupID:     grpEng.ID,
		Round:       1,
		LeaderID:    &leaderUser.ID,
		Phase:       "initial",
		Status:      "approved",
		Vote:        "approved",
		RespondedAt: &respTime,
	})
	_ = tx.Where("draft_id = ? AND group_id = ?", d4Batch.ID, grpQA.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d4Batch.ID,
		GroupID:     grpQA.ID,
		Round:       1,
		LeaderID:    &leaderQA.ID,
		Phase:       "initial",
		Status:      "approved",
		Vote:        "approved",
		RespondedAt: &respTime,
	})

	// Checkpoints for Batch D4 (500 committed, 500 committed, 1 failed)
	_ = tx.Where("draft_id = ? AND batch_number = ?", d4Batch.ID, 1).FirstOrCreate(&model.DraftCheckpoint{
		DraftID:      d4Batch.ID,
		ExecutionID:  4,
		BatchNumber:  1,
		CursorLastID: "500",
		RowsAffected: 500,
		Status:       "committed",
		CreatedAt:    now.Add(-20 * time.Minute),
	})
	_ = tx.Where("draft_id = ? AND batch_number = ?", d4Batch.ID, 2).FirstOrCreate(&model.DraftCheckpoint{
		DraftID:      d4Batch.ID,
		ExecutionID:  4,
		BatchNumber:  2,
		CursorLastID: "1000",
		RowsAffected: 500,
		Status:       "committed",
		CreatedAt:    now.Add(-15 * time.Minute),
	})
	_ = tx.Where("draft_id = ? AND batch_number = ?", d4Batch.ID, 3).FirstOrCreate(&model.DraftCheckpoint{
		DraftID:      d4Batch.ID,
		ExecutionID:  4,
		BatchNumber:  3,
		CursorLastID: "1500",
		RowsAffected: 0,
		Status:       "failed",
		CreatedAt:    now.Add(-10 * time.Minute),
	})

	// --- DRAFT 5: Rejected / Vetoed DDL on db2_mysql ---
	d5Veto := model.Draft{
		DatabaseID:         db2.ID,
		UserID:             userUser.ID,
		SqlUp:              "DROP TABLE inventory_movements;",
		Status:             "rejected",
		CurrentRound:       1,
		CreatedAt:          now.Add(-3 * time.Hour),
		UpdatedAt:          now.Add(-2 * time.Hour),
	}
	if err := tx.Where("database_id = ? AND sql_up LIKE ?", d5Veto.DatabaseID, "DROP TABLE inventory_movements%").FirstOrCreate(&d5Veto).Error; err != nil {
		return err
	}

	_ = tx.Where("draft_id = ? AND group_id = ?", d5Veto.ID, grpEng.ID).FirstOrCreate(&model.DraftApproval{
		DraftID:     d5Veto.ID,
		GroupID:     grpEng.ID,
		Round:       1,
		LeaderID:    &leaderUser.ID,
		Phase:       "initial",
		Status:      "rejected",
		Vote:        "rejected",
		RespondedAt: &respTime,
	})

	// --- DRAFT 6: Execution Failed Draft on db4_billing (Tests 'failed' filter) ---
	d6Failed := model.Draft{
		DatabaseID:         db4.ID,
		UserID:             userUser.ID,
		SqlUp:              "ALTER TABLE invoices ADD CONSTRAINT chk_amount_positive CHECK (amount > 999999999999);",
		Status:             "failed",
		CurrentRound:       1,
		CreatedAt:          now.Add(-4 * time.Hour),
		UpdatedAt:          now.Add(-3 * time.Hour),
	}
	_ = tx.Where("database_id = ? AND sql_up LIKE ?", db4.ID, "%chk_amount_positive%").FirstOrCreate(&d6Failed)

	return nil
}

func databaseKey(name string) string {
	return fmt.Sprintf("name:%s", name)
}

func Seed(db *gorm.DB) error {
	return SeedTestData(db)
}
