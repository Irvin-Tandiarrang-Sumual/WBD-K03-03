package domain

type Database struct {
	ID         uint
	Port       uint
	Host       string
	User       string
	Password   string
	Type       string
	Name       string
	AccessType string
}

type CreateDatabaseInput struct {
	Port     uint
	Host     string
	User     string
	Password string
	Type     string
	Name     string
}
