package service

import (
	"errors"
	"strings"
	"time"

	"automarket/auth"
	"automarket/models"
	"automarket/storage"
)

type Accounts struct {
	users    *storage.Table[models.User]
	vehicles *storage.Table[models.Vehicle]
	sessions *auth.Sessions
}

func NewAccounts(users *storage.Table[models.User], vehicles *storage.Table[models.Vehicle], sessions *auth.Sessions) *Accounts {
	return &Accounts{users: users, vehicles: vehicles, sessions: sessions}
}

type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

// Register crea la cuenta de un vendedor (HU-VEN-01, registro autónomo).
func (a *Accounts) Register(in RegisterInput) (models.User, error) {
	return a.create(in, models.RoleSeller)
}

// EnsureAdmin crea la cuenta de administrador indicada si aún no existe.
// El sistema no expone ningún endpoint para crear administradores.
func (a *Accounts) EnsureAdmin(in RegisterInput) error {
	_, err := a.create(in, models.RoleAdmin)
	if errors.Is(err, ErrEmailTaken) {
		return nil
	}
	return err
}

// Login valida las credenciales y emite un token de sesión.
func (a *Accounts) Login(email, password string) (models.User, string, error) {
	email = normalizeEmail(email)
	user, err := a.users.Find(func(u models.User) bool { return u.Email == email })
	if err != nil || !auth.VerifyPassword(password, user.PasswordHash) {
		return models.User{}, "", ErrInvalidCredentials
	}
	return user, a.sessions.Create(user.ID), nil
}

// Authenticate resuelve el usuario dueño de un token. Si el usuario fue
// eliminado, su token deja de ser válido.
func (a *Accounts) Authenticate(token string) (models.User, error) {
	userID, ok := a.sessions.Resolve(token)
	if !ok {
		return models.User{}, ErrInvalidCredentials
	}
	user, err := a.users.Get(userID)
	if err != nil {
		return models.User{}, ErrInvalidCredentials
	}
	return user, nil
}

// Delete elimina un vendedor junto con sus publicaciones (HU-ADM-01).
func (a *Accounts) Delete(userID int) error {
	user, err := a.users.Get(userID)
	if err != nil {
		return translate(err)
	}
	if user.Role != models.RoleSeller {
		return ErrInvalidState
	}
	if err := a.vehicles.DeleteWhere(func(v models.Vehicle) bool { return v.SellerID == userID }); err != nil {
		return err
	}
	return translate(a.users.Delete(userID))
}

func (a *Accounts) create(in RegisterInput, role models.Role) (models.User, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return models.User{}, err
	}
	email := normalizeEmail(in.Email)

	user, err := a.users.Insert(func(id int) models.User {
		return models.User{
			ID:           id,
			Name:         strings.TrimSpace(in.Name),
			Email:        email,
			Phone:        strings.TrimSpace(in.Phone),
			Role:         role,
			PasswordHash: hash,
			CreatedAt:    time.Now().UTC(),
		}
	}, func(u models.User) bool { return u.Email == email })
	if errors.Is(err, storage.ErrConflict) {
		return models.User{}, ErrEmailTaken
	}
	return user, err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
