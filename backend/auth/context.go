package auth

import (
	"automarket/models"

	"github.com/gin-gonic/gin"
)

const (
	actorKey = "auth.actor"
	emailKey = "auth.email"

	// maxEmailLength limita el correo registrado, que el cliente controla.
	maxEmailLength = 254
)

// SetActor asocia al request el usuario que lo ejecuta, para que la
// autorización y la auditoría puedan consultarlo.
func SetActor(c *gin.Context, user models.User) {
	c.Set(actorKey, user)
}

// Actor devuelve el usuario asociado al request; false si es un visitante.
func Actor(c *gin.Context) (models.User, bool) {
	value, ok := c.Get(actorKey)
	if !ok {
		return models.User{}, false
	}
	user, ok := value.(models.User)
	return user, ok
}

// SetEmail asocia al request el correo con que el cliente intentó
// autenticarse o registrarse, para que Accounting lo registre.
func SetEmail(c *gin.Context, email string) {
	if len(email) > maxEmailLength {
		email = email[:maxEmailLength]
	}
	c.Set(emailKey, email)
}

// Email devuelve el correo asociado al request, o "" si no hay.
func Email(c *gin.Context) string {
	return c.GetString(emailKey)
}
