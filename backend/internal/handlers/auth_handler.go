package handlers

import (
	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AuthHandler struct{ service *services.AuthService }

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}
func (h *AuthHandler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.AUTH01001, "Enter your email and password."))
		return
	}
	token, user, err := h.service.Login(request.Email, request.Password)
	if err != nil {
		middleware.JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01001, "Invalid email or password."))
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"token": token, "user": user}))
}
func (h *AuthHandler) Logout(c *gin.Context) {
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"loggedOut": true}))
}
