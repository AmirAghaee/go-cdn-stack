package http

import (
	"context"
	"control-panel/internal/helper"
	"errors"
	"net/http"

	"control-panel/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cdnService  service.CdnServiceInterface
	userService service.UserServiceInterface
}

func NewHTTPHandler(cdnService service.CdnServiceInterface, userService service.UserServiceInterface) *Handler {
	return &Handler{
		cdnService:  cdnService,
		userService: userService,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	// CDN routes
	r.POST("/cdns", h.createCDN)
	r.GET("/cdns", h.listCDNs)
	r.GET("/cdns/:id", h.getCDN)
	r.PUT("/cdns/:id", h.updateCDN)
	r.DELETE("/cdns/:id", h.deleteCDN)

	// User routes
	r.POST("/users", h.createUser)
	r.GET("/users", h.listUsers)
	r.POST("/login", h.loginUser)
}

// ================== CDN Handlers ==================

func (h *Handler) createCDN(c *gin.Context) {
	var body struct {
		Origin   string `json:"origin" binding:"required,url"`
		Domain   string `json:"domain" binding:"required"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sErr := helper.ErrInvalidInput()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}

	if err := h.cdnService.Create(context.Background(), body.Origin, body.Domain, body.IsActive); err != nil {
		var sErr *helper.ServiceError
		if errors.As(err, &sErr) {
			c.JSON(sErr.Code, gin.H{"error": sErr.Message})
			return
		}
		// fallback unexpected error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) listCDNs(c *gin.Context) {
	cdns, _ := h.cdnService.List(context.Background())
	c.JSON(http.StatusOK, cdns)
}

func (h *Handler) getCDN(c *gin.Context) {
	id := c.Param("id")
	cdn, err := h.cdnService.Get(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, cdn)
}

func (h *Handler) updateCDN(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Origin   string `json:"origin" binding:"required,url"`
		Domain   string `json:"domain" binding:"required"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.cdnService.Update(context.Background(), id, body.Origin, body.Domain, body.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteCDN(c *gin.Context) {
	id := c.Param("id")
	if err := h.cdnService.Delete(context.Background(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ================== User Handlers ==================

func (h *Handler) createUser(c *gin.Context) {
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
	c.Status(http.StatusCreated)
}

func (h *Handler) listUsers(c *gin.Context) {
	users, _ := h.userService.List(context.Background())
	c.JSON(http.StatusOK, users)
}

func (h *Handler) loginUser(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sErr := helper.ErrInvalidInput()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}
	user, err := h.userService.Login(context.Background(), body.Email, body.Password)
	if err != nil {
		sErr := helper.ErrUnAuthorized()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}
	c.JSON(http.StatusOK, user)
}
