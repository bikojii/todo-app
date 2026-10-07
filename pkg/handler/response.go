package handler

import (
	"errors"
	"net/http"

	todo "github.com/bikojii/todo-app"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type errorResponse struct {
	Message string `json:"message"`
}
type statusResponse struct {
	Status string `json:"status"`
}

func newErrorResponse(c *gin.Context, statusCode int, message string) {
	c.AbortWithStatusJSON(statusCode, errorResponse{message})
}

func serviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, todo.ErrInvalidInput):
		newErrorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, todo.ErrNotFound):
		newErrorResponse(c, http.StatusNotFound, todo.ErrNotFound.Error())
	case errors.Is(err, todo.ErrConflict):
		newErrorResponse(c, http.StatusConflict, todo.ErrConflict.Error())
	case errors.Is(err, todo.ErrCredentials), errors.Is(err, todo.ErrToken):
		newErrorResponse(c, http.StatusUnauthorized, "invalid credentials or token")
	default:
		logrus.WithError(err).Error("request failed")
		newErrorResponse(c, http.StatusInternalServerError, "internal server error")
	}
}
