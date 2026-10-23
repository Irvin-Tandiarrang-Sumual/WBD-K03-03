package app

import (
	"fmt"
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/postgresql/system"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations"
)

func Run() {
	fmt.Println("Hello, world!")

	config, _ := config.LoadConfig()

	db, err := system.ConnectToSystemDatabase(config.Database)

	if err != nil {
		log.Printf("Failed to connect to db")
		return
	}

	if err := migrations.Migrate(db); err != nil {
		log.Printf("Failed to migrate db")
		return
	}

	log.Print("Hi!")
}
