package httpapi

import (
	"errors"
	"log"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *identity.Service
}

func New(service *identity.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublic(routes *gin.Engine) {
	routes.POST("/login", h.login)
}

func (h *Handler) RegisterProtected(routes *gin.RouterGroup) {
	routes.POST("/register", h.register)
	routes.GET("/users", h.list)
	routes.PUT("/users/:id/password", h.changePassword)
	routes.DELETE("/users/:id", h.delete)
}

func (h *Handler) register(c *gin.Context) {
	var body credentialsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inputs"})
		return
	}

	if err := h.service.Register(c.Request.Context(), body.Email, body.Password); err != nil {
		if errors.Is(err, identity.ErrUserExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "user created successfully"})
}

func (h *Handler) login(c *gin.Context) {
	var body credentialsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inputs"})
		return
	}

	result, err := h.service.Login(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, loginResponse{Token: result.Token, User: newUserResponse(&result.User)})
}

func (h *Handler) list(c *gin.Context) {
	users, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []userResponse
	for _, user := range users {
		response = append(response, newUserResponse(user))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) changePassword(c *gin.Context) {
	var body passwordRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inputs"})
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), c.Param("id"), body.Password); err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		log.Printf("failed to change user password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

func (h *Handler) delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrSelfDelete):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, identity.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			log.Printf("failed to delete user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		}
		return
	}
	c.Status(http.StatusNoContent)
}
