package domain

import (
	"time"
)

const (
	MigrationStatusSuccess = "success"
	MigrationStatusFail    = "fail"
)

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
	DatabaseID uint
	UserID     uint
	SqlUp      string
}

type MigrationComparison struct {
	FirstMigration  *Migration
	SecondMigration *Migration
	SQLUpEqual      bool
	SQLDownEqual    bool
}

func (d *CreateMigrationInput) IsValid() error {
	if d.DatabaseID == 0 || d.UserID == 0 || d.SqlUp == "" {
		return ErrInvalidMigrationInput
	}
	return nil
}

type CompareMigrationsInput struct {
	DatabaseID        uint
	UserID            uint
	FirstMigrationID  uint
	SecondMigrationID uint
}

func (d *CompareMigrationsInput) IsValid() error {
	if d.DatabaseID == 0 || d.UserID == 0 || d.FirstMigrationID == 0 || d.SecondMigrationID == 0 || d.FirstMigrationID == d.SecondMigrationID {
		return ErrInvalidMigrationComparison
	}
	return nil
}

type RollbackMigrationsInput struct {
	DatabaseID        uint
	UserID            uint
	TargetMigrationID uint
}

func (d *RollbackMigrationsInput) IsValid() error {
	if d.DatabaseID == 0 || d.UserID == 0 || d.TargetMigrationID == 0 {
		return ErrInvalidMigrationInput
	}
	return nil
}
