package http

import (
	"context"
	"control-panel/internal/helper"
	"control-panel/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CdnHandler struct {
	cdnService service.CdnServiceInterface
}

func NewCdnHandler(cdnService service.CdnServiceInterface) *CdnHandler {
	return &CdnHandler{cdnService: cdnService}
}

func (h *CdnHandler) Register(r *gin.Engine) {
	r.POST("/cdns", h.createCDN)
	r.GET("/cdns", h.listCDNs)
	r.GET("/cdns/:id", h.getCDN)
	r.PUT("/cdns/:id", h.updateCDN)
	r.DELETE("/cdns/:id", h.deleteCDN)
}

func (h *CdnHandler) createCDN(c *gin.Context) {
	var body struct {
		Origin   string `json:"origin" binding:"required,url"`
		Domain   string `json:"domain" binding:"required"`
		IsActive bool   `json:"is_active"`
		CacheTTL uint   `json:"cache_ttl"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sErr := helper.ErrInvalidInput()
		c.JSON(sErr.Code, gin.H{"error": sErr.Message})
		return
	}

	if err := h.cdnService.Create(context.Background(), body.Origin, body.Domain, body.IsActive, body.CacheTTL); err != nil {
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

func (h *CdnHandler) listCDNs(c *gin.Context) {
	cdns, _ := h.cdnService.List(context.Background())
	c.JSON(http.StatusOK, cdns)
}

func (h *CdnHandler) getCDN(c *gin.Context) {
	id := c.Param("id")
	cdn, err := h.cdnService.Get(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, cdn)
}

func (h *CdnHandler) updateCDN(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Origin   string `json:"origin" binding:"required,url"`
		Domain   string `json:"domain" binding:"required"`
		IsActive bool   `json:"is_active"`
		CacheTTL uint   `json:"cache_ttl"`
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

func (h *CdnHandler) deleteCDN(c *gin.Context) {
	id := c.Param("id")
	if err := h.cdnService.Delete(context.Background(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
