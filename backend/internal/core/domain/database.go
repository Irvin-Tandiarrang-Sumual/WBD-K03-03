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

type CreateDatabaseInput struct {
	Port     uint
	Host     string
	User     string
	Password string
	Type     string
	Name     string
}

type DatabaseHost struct {
	Host string `json:"host"`
	Port uint   `json:"port"`
	User string `json:"user"`
	Type string `json:"type"`
}

