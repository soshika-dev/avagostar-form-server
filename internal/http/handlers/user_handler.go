package handlers

import (
	"net/http"
	"strings"

	"avagostar-form-server/internal/services"
	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	auth *services.AuthService
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

func NewUserHandler(auth *services.AuthService) *UserHandler {
	return &UserHandler{auth: auth}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "user"
	}

	user, err := h.auth.CreateUser(c.Request.Context(), req.Username, req.Password, role)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	utils.RespondCreated(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}
