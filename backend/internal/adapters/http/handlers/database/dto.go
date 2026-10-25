package database

type CreateDatabaseRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	Type     string `json:"type" binding:"required,oneof=postgresql mysql"`
	Name     string `json:"name" binding:"required"`
}

type DatabaseResponse struct {
	ID         uint   `json:"id"`
	Port       uint   `json:"port"`
	Host       string `json:"host"`
	User       string `json:"user"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	AccessType string `json:"access_type,omitempty"`
}
