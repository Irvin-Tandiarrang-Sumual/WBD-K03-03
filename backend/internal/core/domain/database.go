package domain

type Database struct {
	ID       uint
	Port     uint
	Host     string
	User     string
	Password string
	Type     string
	Name     string
}
