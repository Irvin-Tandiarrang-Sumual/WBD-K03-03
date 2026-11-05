package domain

const (
	DatabaseTypePostgreSQL = "postgresql"
	DatabaseTypeMySQL      = "mysql"
	MaxPortNumber          = 65535
)

type Database struct {
	ID         uint
	Port       uint
	Host       string
	User       string
	Password   string
	Type       string
	Name       string
	AccessType AccessType
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

func (d Database) Validate() error {
	if d.Port == 0 || d.Port > MaxPortNumber {
		return ErrInvalidInput
	}
	if d.Host == "" || d.User == "" || d.Name == "" {
		return ErrInvalidInput
	}
	if !IsValidDatabaseType(d.Type) {
		return ErrInvalidInput
	}
	return nil
}

func IsValidDatabaseType(dbType string) bool {
	return dbType == DatabaseTypePostgreSQL || dbType == DatabaseTypeMySQL
}
