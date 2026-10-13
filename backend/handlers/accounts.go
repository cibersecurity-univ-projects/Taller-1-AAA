package handlers

import (
	"net/http"

	"automarket/auth"
	"automarket/service"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	accounts *service.Accounts
}

func NewAccountHandler(accounts *service.Accounts) *AccountHandler {
	return &AccountHandler{accounts: accounts}
}

type registerRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Phone    string `json:"phone" binding:"required,max=20"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register: POST /auth/register
func (h *AccountHandler) Register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.accounts.Register(service.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	auth.SetActor(c, user)
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "email": user.Email, "role": user.Role})
}

// Login: POST /auth/login
func (h *AccountHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	user, token, err := h.accounts.Login(req.Email, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	auth.SetActor(c, user)
	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role})
}

// DeleteUser: DELETE /users/:id (administrador)
func (h *AccountHandler) DeleteUser(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.accounts.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
