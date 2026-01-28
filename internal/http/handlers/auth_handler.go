package handlers

import (
	"net/http"

	"avagostar-form-server/internal/services"
	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *services.AuthService
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotRequest struct {
	Username string `json:"username" binding:"required"`
}

type ResetRequest struct {
	Username    string `json:"username" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required"`
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Forgot(c *gin.Context) {
	var req ForgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	resp, err := h.auth.ForgotPassword(c.Request.Context(), req.Username)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Reset(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if err := h.auth.ResetPassword(c.Request.Context(), req.Username, req.Code, req.NewPassword); err != nil {
		utils.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
