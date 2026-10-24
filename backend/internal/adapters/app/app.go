package app

import (
	"fmt"
	"log"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/router"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/storage/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/migrations"
	"github.com/gin-gonic/gin"
)

func Run() {
	fmt.Println("Hello, world!")

	config, _ := config.LoadConfig()

	db, err := postgresql.ConnectToSystemDatabase(config.Database)

	if err != nil {
		log.Printf("Failed to connect to db")
		return
	}

	if err := migrations.Migrate(db); err != nil {
		log.Printf("Failed to migrate db")
		return
	}

	log.Print("Hi!")

	r := gin.Default()

	router.SetUpRouter(r)
}
