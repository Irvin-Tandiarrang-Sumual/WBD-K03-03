package domain

import "time"

type Migration struct {
	ID         uint
	DatabaseID uint
	Version    string
	MigratedAt time.Time
	Status     string
	SqlUp      string
	SqlDown    string
	Snapshot   []byte
	UserID     uint
}
