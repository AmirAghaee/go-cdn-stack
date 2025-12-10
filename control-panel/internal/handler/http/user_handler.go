package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/helper"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService service.UserServiceInterface
}

func NewUserHandler(userService service.UserServiceInterface) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) Register(g *gin.Engine, protected *gin.RouterGroup) {
	// Public routes
	g.POST("/register", h.createUser)
	g.POST("/login", h.loginUser)

	// Protected routes
	protected.GET("/users", h.listUsers)
}

func (h *UserHandler) createUser(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sErr := helper.ErrInvalidInput()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}
	if err := h.userService.Register(context.Background(), body.Email, body.Password); err != nil {
		var sErr *helper.ServiceError
		if errors.As(err, &sErr) {
			c.JSON(sErr.Code, gin.H{"error": sErr.Message})
			return
		}
		// fallback unexpected error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "user created successfully"})
}

func (h *UserHandler) listUsers(c *gin.Context) {
	users, _ := h.userService.List(context.Background())
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) loginUser(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sErr := helper.ErrInvalidInput()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}
	response, err := h.userService.Login(context.Background(), body.Email, body.Password)
	if err != nil {
		sErr := helper.ErrUnAuthorized()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}
	c.JSON(http.StatusOK, response)
}
