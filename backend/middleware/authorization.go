package middleware

import (
	"net/http"
	"slices"

	"automarket/auth"
	"automarket/models"

	"github.com/gin-gonic/gin"
)

// RequireRole permite continuar solo si el solicitante autenticado tiene
// alguno de los roles indicados (Authorization). Debe ir después de
// Authenticate.
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.Actor(c)
		if !ok {
			reject(c, http.StatusUnauthorized, "se requiere autenticación")
			return
		}
		if !slices.Contains(roles, user.Role) {
			reject(c, http.StatusForbidden, "el rol no tiene permiso para esta operación")
			return
		}
		c.Next()
	}
}
