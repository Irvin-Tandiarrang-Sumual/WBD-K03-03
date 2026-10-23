package migration

import "time"

type Migration struct {
	DatabaseID uint
	Version    string
	MigratedAt time.Time
	Status     string
	SqlUp      string
	SqlDown    string
	Snapshot   []byte
	UserID     uint
}
