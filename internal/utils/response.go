package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type AppError struct {
	Status  int
	Code    string
	Message string
	Details interface{}
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(status int, code, message string, details interface{}) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

func RespondError(c *gin.Context, err error) {
	appErr, ok := err.(*AppError)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorBody{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		}})
		return
	}

	c.JSON(appErr.Status, ErrorResponse{Error: ErrorBody{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}})
}

func RespondValidationError(c *gin.Context, details interface{}) {
	c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorBody{
		Code:    "VALIDATION_ERROR",
		Message: "invalid request",
		Details: details,
	}})
}

func RespondOK(c *gin.Context, payload interface{}) {
	c.JSON(http.StatusOK, payload)
}

func RespondCreated(c *gin.Context, payload interface{}) {
	c.JSON(http.StatusCreated, payload)
}
