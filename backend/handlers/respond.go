// Package handlers traduce las solicitudes HTTP a llamadas a los servicios
// y sus resultados a respuestas JSON.
package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"automarket/service"

	"github.com/gin-gonic/gin"
)

// errorStatus asocia cada error de dominio a su código HTTP.
var errorStatus = map[error]int{
	service.ErrNotFound:           http.StatusNotFound,
	service.ErrEmailTaken:         http.StatusConflict,
	service.ErrInvalidCredentials: http.StatusUnauthorized,
	service.ErrNotOwner:           http.StatusForbidden,
	service.ErrInvalidState:       http.StatusConflict,
}

// respondError responde con el código asociado al error. Los errores no
// previstos se registran y se ocultan al cliente.
func respondError(c *gin.Context, err error) {
	for known, status := range errorStatus {
		if errors.Is(err, known) {
			c.JSON(status, gin.H{"error": known.Error()})
			return
		}
	}
	log.Printf("error interno en %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
}

// bindJSON valida el cuerpo contra las reglas `binding` del destino y
// responde 400 si no las cumple.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return false
	}
	return true
}

// pathID lee el parámetro :id. Un id mal formado se trata como inexistente.
func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		respondError(c, service.ErrNotFound)
		return 0, false
	}
	return id, true
}
