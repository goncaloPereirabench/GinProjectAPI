package http

import (
	"net/http"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

type authRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type authResponse struct {
	User  userResponse  `json:"user"`
	Token tokenResponse `json:"token"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, err)
		return
	}

	user, token, err := h.auth.Register(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusCreated, newAuthResponse(user, token))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, err)
		return
	}

	user, token, err := h.auth.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, newAuthResponse(user, token))
}

func newAuthResponse(user domain.User, token service.AuthToken) authResponse {
	return authResponse{
		User: userResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  user.Role,
		},
		Token: tokenResponse{
			AccessToken: token.AccessToken,
			TokenType:   token.TokenType,
			ExpiresAt:   token.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
}
