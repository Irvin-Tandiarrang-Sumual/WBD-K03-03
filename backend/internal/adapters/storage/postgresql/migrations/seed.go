package migrations

import (
	"errors"
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		databases, err := seedDatabases(tx)
		if err != nil {
			return err
		}

		groups, err := seedGroups(tx, databases)
		if err != nil {
			return err
		}

		return seedUsers(tx, groups)
	})
}

func seedDatabases(tx *gorm.DB) (map[string]model.Database, error) {
	databaseSeeds := []model.Database{
		{Port: 5432, Host: "dummy-postgres", User: "demo_user", Password: "demo_password", Type: "postgresql", Name: "db1_pg"},
		{Port: 3306, Host: "dummy-mysql", User: "demo_user", Password: "demo_password", Type: "mysql", Name: "db2_mysql"},
	}

	databases := make(map[string]model.Database, len(databaseSeeds))
	for _, databaseSeed := range databaseSeeds {
		var database model.Database
		err := tx.Where("port = ?", databaseSeed.Port).First(&database).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			database = databaseSeed
			err = tx.Create(&database).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding database on port %d: %w", databaseSeed.Port, err)
		}
		databases[databaseKey(databaseSeed.Port)] = database
	}
	return databases, nil
}

func seedGroups(tx *gorm.DB, databases map[string]model.Database) (map[string]model.Group, error) {
	groupSeeds := []struct {
		key        string
		name       string
		accessType string
		databases  []model.Database
	}{
		{"admin", "admin", "view_edit", []model.Database{{ID: databases[databaseKey(5432)].ID}, {ID: databases[databaseKey(3306)].ID}}},
		{"user", "user", "view_only", []model.Database{{ID: databases[databaseKey(5432)].ID}, {ID: databases[databaseKey(3306)].ID}}},
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

func seedUsers(tx *gorm.DB, groups map[string]model.Group) error {
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

func databaseKey(port uint) string {
	return fmt.Sprintf("port:%d", port)
}
