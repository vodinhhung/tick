package handler

import (
	"github.com/gin-gonic/gin"
)

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondError(c *gin.Context, httpStatus int, code string, message string) {
	c.JSON(httpStatus, ApiError{
		Code:    code,
		Message: message,
	})
}

func getUserID(c *gin.Context) uint {
	val, exists := c.Get("userID")
	if !exists {
		return 0
	}
	return val.(uint)
}
