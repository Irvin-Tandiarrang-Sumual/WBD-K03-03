package database

import "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"

type CreateDatabaseRequest struct {
	Port     uint   `json:"port" binding:"required"`
	Host     string `json:"host" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	Type     string `json:"type"`
	Name     string `json:"name" binding:"required"`
}

type DatabaseResponse struct {
	ID   uint   `json:"id"`
	Port uint   `json:"port"`
	Host string `json:"host"`
	User string `json:"user"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func toDatabaseResponse(database domain.Database) DatabaseResponse {
	return DatabaseResponse{
		ID:   database.ID,
		Port: database.Port,
		Host: database.Host,
		User: database.User,
		Type: database.Type,
		Name: database.Name,
	}
}