package handler

import (
	"net/http"

	"github.com/bikojii/todo-app"
	"github.com/gin-gonic/gin"
)

func (h *Handler) signUp(c *gin.Context) {
	var input todo.User

	if err := bindJSON(c, &input); err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.services.Authorization.CreateUser(input)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, map[string]interface{}{
		"id": id,
	})

}

type signInInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) signIn(c *gin.Context) {
	var input signInInput

	if err := bindJSON(c, &input); err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.services.Authorization.GenerateToken(input.Username, input.Password)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
	})

}
