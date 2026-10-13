// Package service contiene las reglas de negocio de AutoMarket. No conoce
// HTTP: reporta los fallos con los errores de este archivo.
package service

import (
	"errors"

	"automarket/storage"
)

var (
	ErrNotFound           = errors.New("recurso no encontrado")
	ErrEmailTaken         = errors.New("el correo ya está registrado")
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrNotOwner           = errors.New("el recurso pertenece a otro usuario")
	ErrInvalidState       = errors.New("la operación no es válida en el estado actual del recurso")
)

// translate convierte los errores de persistencia en errores del dominio.
func translate(err error) error {
	if errors.Is(err, storage.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
