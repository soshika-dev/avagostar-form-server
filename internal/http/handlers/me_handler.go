package handlers

import (
	"net/http"

	"avagostar-form-server/internal/repo"
	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	users *repo.UserRepo
}

func NewMeHandler(users *repo.UserRepo) *MeHandler {
	return &MeHandler{users: users}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		utils.RespondError(c, utils.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "missing user", nil))
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		utils.RespondError(c, utils.NewAppError(http.StatusNotFound, "NOT_FOUND", "user not found", nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}
