package domain

import "time"

type Migration struct {
	ID           uint
	DatabaseID   uint
	Version      string
	MigratedAt   time.Time
	Status       string
	SqlUp        string
	SqlDown      string
	ErrorMessage string
	Snapshot     []byte
	UserID       uint
}

type CreateMigrationInput struct {
	databaseID uint
	userID     uint
	sqlUp      string
}

type CompareMigrationsInput struct {
	DatabaseID        uint
	UserID            uint
	FirstMigrationID  uint
	SecondMigrationID uint
}

type RollbackMigrationsInput struct {
	databaseID        uint
	userID            uint
	targetMigrationID uint
}
