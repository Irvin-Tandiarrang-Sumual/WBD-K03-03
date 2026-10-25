// @title			WBD K03-03 API
// @version			1.0
// @description		Backend API for WBD K03-03 — database management platform with user/group access control and migration support.
// @host			localhost:8080
// @BasePath		/api/v1
// @schemes			http https
//
// @securityDefinitions.apikey	CookieAuth
// @in							header
// @name						Cookie
// @description					Session cookie. After login, copy the value of the `session` cookie from the Set-Cookie header and paste it here as: `session=<token>`
package main

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/app"
)

func main() {
	app.Run()
}
