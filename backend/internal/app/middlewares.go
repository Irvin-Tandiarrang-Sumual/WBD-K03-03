package app

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type applicationMiddlewares struct {
	authMiddleware *middleware.AuthMiddleware
}

func buildMiddlewares(sessionsServices ports.SessionService,
	userServices ports.UserService, isSecureCookie bool) *applicationMiddlewares {
	return &applicationMiddlewares{
		authMiddleware: middleware.NewAuthMiddleware(sessionsServices,
			userServices, isSecureCookie),
	}
}
