package httpapi

import (
	"errors"
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
	routes.POST("/register", h.register)
	routes.POST("/login", h.login)
}

func (h *Handler) RegisterProtected(routes *gin.RouterGroup) {
	routes.GET("/users", h.list)
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
