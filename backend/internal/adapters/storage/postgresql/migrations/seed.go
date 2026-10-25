package migrations

import (
	"errors"
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
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

		groups, err := seedGroups(tx, databases)
		if err != nil {
			return err
		}

		return seedTestUsers(tx, groups)
	})
}

func seedTestDatabases(tx *gorm.DB, encodePassword func(string) (string, error)) (map[string]model.Database, error) {
	databaseSeeds := []model.Database{
		{Port: 5432, Host: "demo_postgres", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db1_pg"},
		{Port: 3306, Host: "demo_mysql", User: "demo_user", Password: "demo_password", Type: "mysql", Name: "db2_mysql"},
		{Port: 5432, Host: "demo_postgres_extra", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db3_pg_empty"},
		{Port: 5432, Host: "demo_postgres_extra", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db4_pg_sales"},
		{Port: 3306, Host: "demo_mysql_extra", User: "demo_user", Password: "demo_password", Type: "mysql", Name: "db5_mysql_analytics"},
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
				"port":     databaseSeed.Port,
				"host":     databaseSeed.Host,
				"user":     databaseSeed.User,
				"password": databaseSeed.Password,
				"type":     databaseSeed.Type,
			}).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding database %s: %w", databaseSeed.Name, err)
		}
		databases[databaseKey(databaseSeed.Name)] = database
	}
	return databases, nil
}

func seedGroups(tx *gorm.DB, databases map[string]model.Database) (map[string]model.Group, error) {
	allDatabases := []model.Database{
		{ID: databases[databaseKey("db1_pg")].ID},
		{ID: databases[databaseKey("db2_mysql")].ID},
		{ID: databases[databaseKey("db3_pg_empty")].ID},
		{ID: databases[databaseKey("db4_pg_sales")].ID},
		{ID: databases[databaseKey("db5_mysql_analytics")].ID},
	}
	groupSeeds := []struct {
		key        string
		name       string
		accessType string
		databases  []model.Database
	}{
		{"admin", "admin", "view_edit", allDatabases},
		{"user", "user", "view_only", allDatabases},
	}

	groups := make(map[string]model.Group, len(groupSeeds))
	for _, groupSeed := range groupSeeds {
		var group model.Group
		err := tx.Where("name = ?", groupSeed.name).First(&group).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			group = model.Group{Name: groupSeed.name}
			err = tx.Create(&group).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding %s group: %w", groupSeed.key, err)
		}

		if err := tx.Where("group_id = ?", group.ID).Delete(&model.GroupDatabase{}).Error; err != nil {
			return nil, fmt.Errorf("clearing %s group databases: %w", groupSeed.key, err)
		}
		for _, database := range groupSeed.databases {
			if err := tx.Create(&model.GroupDatabase{
				GroupID: group.ID, DatabaseID: database.ID, AccessType: groupSeed.accessType,
			}).Error; err != nil {
				return nil, fmt.Errorf("seeding %s group database: %w", groupSeed.key, err)
			}
		}
		groups[groupSeed.key] = group
	}
	return groups, nil
}

func seedTestUsers(tx *gorm.DB, groups map[string]model.Group) error {
	userSeeds := []struct {
		username string
		password string
		role     string
		group    model.Group
	}{
		{"admin", "admin123", "admin", groups["admin"]},
		{"user", "user123", "user", groups["user"]},
	}

	for _, userSeed := range userSeeds {
		var user model.User
		err := tx.Where("username = ?", userSeed.username).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			password, hashErr := bcrypt.GenerateFromPassword([]byte(userSeed.password), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("hashing password for %s: %w", userSeed.username, hashErr)
			}
			user = model.User{
				Username: userSeed.username,
				Password: string(password),
				Role:     userSeed.role,
				Groups:   []model.Group{{ID: userSeed.group.ID}},
			}
			err = tx.Create(&user).Error
		}
		if err != nil {
			return fmt.Errorf("seeding user %s: %w", userSeed.username, err)
		}
		if err := tx.Model(&user).Association("Groups").Replace([]model.Group{{ID: userSeed.group.ID}}); err != nil {
			return fmt.Errorf("syncing groups for %s user: %w", userSeed.username, err)
		}
	}
	return nil
}

func databaseKey(name string) string {
	return fmt.Sprintf("name:%s", name)
}

func Seed(db *gorm.DB) error {
	return SeedTestData(db)
}
