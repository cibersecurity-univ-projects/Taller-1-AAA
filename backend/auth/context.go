package auth

import (
	"automarket/models"

	"github.com/gin-gonic/gin"
)

const actorKey = "auth.actor"

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
