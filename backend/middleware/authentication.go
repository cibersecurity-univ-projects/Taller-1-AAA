// Package middleware aplica el marco AAA a las rutas de la API.
package middleware

import (
	"net/http"
	"strings"

	"automarket/auth"
	"automarket/service"

	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer "

// Authenticate exige un token de sesión válido en el encabezado
// Authorization antes de procesar la solicitud (Authentication).
func Authenticate(accounts *service.Accounts) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, found := strings.CutPrefix(header, bearerPrefix)
		if !found || token == "" {
			reject(c, http.StatusUnauthorized, "se requiere autenticación")
			return
		}

		user, err := accounts.Authenticate(token)
		if err != nil {
			reject(c, http.StatusUnauthorized, "token inválido o expirado")
			return
		}
		auth.SetActor(c, user)
		c.Next()
	}
}

func reject(c *gin.Context, status int, message string) {
	if status == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", "Bearer")
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}
