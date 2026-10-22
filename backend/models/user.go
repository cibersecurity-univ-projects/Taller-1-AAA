package models

import "time"

// Role identifica el rol de un usuario registrado. El visitante no tiene cuenta:
// solo se usa para identificar a quien opera sin sesión (Accounting).
type Role string

const (
	RoleSeller  Role = "vendedor"
	RoleAdmin   Role = "administrador"
	RoleVisitor Role = "visitante"
)

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

// Contact es la información del vendedor que se expone públicamente.
type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (u User) Contact() Contact {
	return Contact{Name: u.Name, Email: u.Email, Phone: u.Phone}
}
