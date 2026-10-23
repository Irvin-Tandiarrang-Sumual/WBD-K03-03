package migrations

import (
	"errors"
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations/models"
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

func seedDatabases(tx *gorm.DB) (map[string]models.Database, error) {
	databaseSeeds := []models.Database{
		{
			Port:     5432,
			Host:     "dummy-postgres",
			User:     "demo_user",
			Password: "demo_password",
			Type:     "postgresql",
			Name:     "db1_pg",
		},
		{
			Port:     3306,
			Host:     "dummy-mysql",
			User:     "demo_user",
			Password: "demo_password",
			Type:     "mysql",
			Name:     "db2_mysql",
		},
	}

	databases := make(map[string]models.Database, len(databaseSeeds))
	for _, databaseSeed := range databaseSeeds {
		var database models.Database
		err := tx.Where("port = ?", databaseSeed.Port).
			First(&database).Error
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

func seedGroups(tx *gorm.DB, databases map[string]models.Database) (map[string]models.Group, error) {
	groupSeeds := []struct {
		key        string
		accessType string
		databases  []models.Database
	}{
		{
			key:        "admin",
			accessType: "view_edit",
			databases: []models.Database{
				databases[databaseKey(5432)],
				databases[databaseKey(3306)],
			},
		},
		{
			key:        "user",
			accessType: "view_only",
			databases: []models.Database{
				databases[databaseKey(5432)],
				databases[databaseKey(3306)],
			},
		},
	}

	groups := make(map[string]models.Group, len(groupSeeds))
	for _, groupSeed := range groupSeeds {
		var group models.Group
		err := tx.Where("access_type = ? AND database_id = ?",
			groupSeed.accessType,
			groupSeed.databases[0].ID,
		).First(&group).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			group = models.Group{
				AccessType: groupSeed.accessType,
				DatabaseID: int64(groupSeed.databases[0].ID),
				Databases:  groupSeed.databases,
			}
			err = tx.Create(&group).Error
		}
		if err != nil {
			return nil, fmt.Errorf("seeding %s group: %w", groupSeed.key, err)
		}

		if err := tx.Model(&group).Association("Databases").Replace(groupSeed.databases); err != nil {
			return nil, fmt.Errorf("syncing databases for %s group: %w", groupSeed.key, err)
		}

		groups[groupSeed.key] = group
	}

	return groups, nil
}

func seedUsers(tx *gorm.DB, groups map[string]models.Group) error {
	userSeeds := []struct {
		username string
		password string
		role     string
		group    models.Group
	}{
		{
			username: "admin",
			password: "admin123",
			role:     "admin",
			group:    groups["admin"],
		},
		{
			username: "user",
			password: "user123",
			role:     "user",
			group:    groups["user"],
		},
	}

	for _, userSeed := range userSeeds {
		var user models.User
		err := tx.Where("username = ?", userSeed.username).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			password, hashErr := bcrypt.GenerateFromPassword(
				[]byte(userSeed.password),
				bcrypt.DefaultCost,
			)
			if hashErr != nil {
				return fmt.Errorf("hashing password for %s: %w", userSeed.username, hashErr)
			}

			user = models.User{
				Username: userSeed.username,
				Password: string(password),
				Role:     userSeed.role,
				Groups:   []models.Group{userSeed.group},
			}
			err = tx.Create(&user).Error
		}
		if err != nil {
			return fmt.Errorf("seeding user %s: %w", userSeed.username, err)
		}

		if err := tx.Model(&user).Association("Groups").Replace([]models.Group{userSeed.group}); err != nil {
			return fmt.Errorf("syncing groups for %s user: %w", userSeed.username, err)
		}
	}

	return nil
}

func databaseKey(port uint) string {
	return fmt.Sprintf("port:%d", port)
}
