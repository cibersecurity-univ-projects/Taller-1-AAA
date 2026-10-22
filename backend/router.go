package main

import (
	"automarket/audit"
	"automarket/handlers"
	"automarket/middleware"
	"automarket/models"
	"automarket/service"

	"github.com/gin-gonic/gin"
)

// newRouter declara las operaciones de la API. Todas pasan por Accounting;
// las protegidas además por Authenticate y RequireRole.
func newRouter(logger *audit.Logger, accounts *service.Accounts, vehicles *service.Vehicles) (*gin.Engine, error) {
	r := gin.New()
	// Sin proxies de confianza, la IP auditada es la de la conexión y no
	// puede falsearse con X-Forwarded-For.
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	r.Use(middleware.Accounting(logger), gin.Recovery())

	accountHandler := handlers.NewAccountHandler(accounts)
	vehicleHandler := handlers.NewVehicleHandler(vehicles)

	// Visitante (HU-VIS-01) y registro/inicio de sesión.
	r.GET("/vehicles", middleware.OptionalAuthenticate(accounts), vehicleHandler.Catalog)
	r.GET("/vehicles/:id", vehicleHandler.Detail)
	r.GET("/vehicles/:id/images/:name", middleware.OptionalAuthenticate(accounts), vehicleHandler.Image)
	r.POST("/auth/register", accountHandler.Register)
	r.POST("/auth/login", accountHandler.Login)

	authenticated := r.Group("", middleware.Authenticate(accounts))

	// Vendedor (HU-VEN-01).
	seller := authenticated.Group("", middleware.RequireRole(models.RoleSeller))
	seller.POST("/vehicles", vehicleHandler.Publish)
	seller.PATCH("/vehicles/:id/sold", vehicleHandler.MarkSold)

	// Administrador (HU-ADM-01).
	admin := authenticated.Group("", middleware.RequireRole(models.RoleAdmin))
	admin.PATCH("/vehicles/:id/approve", vehicleHandler.Approve)
	admin.DELETE("/vehicles/:id", vehicleHandler.Delete)
	admin.DELETE("/users/:id", accountHandler.DeleteUser)

	return r, nil
}
