package httpapi

import (
	"errors"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *cdn.Service
}

func New(service *cdn.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(routes *gin.RouterGroup) {
	routes.POST("/cdns", h.create)
	routes.GET("/cdns", h.list)
	routes.GET("/cdns/:id", h.get)
	routes.PUT("/cdns/:id", h.update)
	routes.DELETE("/cdns/:id", h.delete)
	routes.POST("/snapshot", h.refresh)
}

func (h *Handler) create(c *gin.Context) {
	var body cdnRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inputs"})
		return
	}

	if err := h.service.Create(c.Request.Context(), body.Origin, body.Domain, body.IsActive, body.CacheTTL); err != nil {
		if errors.Is(err, cdn.ErrExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list CDNs"})
		return
	}

	var response []cdnResponse
	for _, item := range items {
		response = append(response, newCDNResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, newCDNResponse(item))
}

func (h *Handler) update(c *gin.Context) {
	var body cdnRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Update(c.Request.Context(), c.Param("id"), body.Origin, body.Domain, body.IsActive, body.CacheTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) refresh(c *gin.Context) {
	if err := h.service.Refresh(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish snapshot"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "snapshot triggered"})
}
