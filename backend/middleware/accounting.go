package middleware

import (
	"log"

	"automarket/audit"
	"automarket/auth"
	"automarket/models"

	"github.com/gin-gonic/gin"
)

// Accounting registra en la bitácora toda solicitud atendida, incluidas las
// públicas y las rechazadas por autenticación o autorización. Debe ser el
// primer middleware para observar el resultado final de cada solicitud.
func Accounting(logger *audit.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		entry := models.AuditEntry{
			ActorRole: string(models.RoleVisitor),
			ClientIP:  c.ClientIP(),
			Method:    c.Request.Method,
			Operation: c.FullPath(),
			Resource:  c.Request.URL.Path,
			Status:    c.Writer.Status(),
		}
		if user, ok := auth.Actor(c); ok {
			entry.ActorID = user.ID
			entry.ActorRole = string(user.Role)
		} else {
			// Sin identidad verificada, el correo intentado es el único
			// rastro de contra qué cuenta iba la solicitud.
			entry.Email = auth.Email(c)
		}
		if err := logger.Record(entry); err != nil {
			log.Printf("auditoría: no se pudo registrar %s %s: %v", entry.Method, entry.Resource, err)
		}
	}
}
