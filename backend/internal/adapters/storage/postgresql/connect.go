package postgresql

import (
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectToSystemDatabase(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
}
