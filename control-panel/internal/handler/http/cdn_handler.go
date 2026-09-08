package http

import (
	"errors"
	"log"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/helper"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"

	"github.com/gin-gonic/gin"
)

type CdnHandler struct {
	cdnService service.CdnServiceInterface
	natsPub    messaging.MessageBrokerInterface
}

func NewCdnHandler(cdnService service.CdnServiceInterface, natsPub messaging.MessageBrokerInterface) *CdnHandler {
	return &CdnHandler{cdnService: cdnService, natsPub: natsPub}
}

func (h *CdnHandler) Register(protected *gin.RouterGroup) {
	protected.POST("/cdns", h.createCDN)
	protected.GET("/cdns", h.listCDNs)
	protected.GET("/cdns/:id", h.getCDN)
	protected.PUT("/cdns/:id", h.updateCDN)
	protected.DELETE("/cdns/:id", h.deleteCDN)
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

	if err := h.cdnService.Create(c.Request.Context(), body.Origin, body.Domain, body.IsActive, body.CacheTTL); err != nil {
		var sErr *helper.ServiceError
		if errors.As(err, &sErr) {
			c.JSON(sErr.Code, gin.H{"error": sErr.Message})
			return
		}
		// fallback unexpected error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.notifySnapshot()
	c.Status(http.StatusCreated)
}

func (h *CdnHandler) listCDNs(c *gin.Context) {
	cdns, err := h.cdnService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list CDNs"})
		return
	}
	c.JSON(http.StatusOK, cdns)
}

func (h *CdnHandler) getCDN(c *gin.Context) {
	id := c.Param("id")
	cdn, err := h.cdnService.Get(c.Request.Context(), id)
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
	if err := h.cdnService.Update(c.Request.Context(), id, body.Origin, body.Domain, body.IsActive, body.CacheTTL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.notifySnapshot()
	c.Status(http.StatusNoContent)
}

func (h *CdnHandler) deleteCDN(c *gin.Context) {
	id := c.Param("id")
	if err := h.cdnService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.notifySnapshot()
	c.Status(http.StatusNoContent)
}

func (h *CdnHandler) notifySnapshot() {
	if err := h.natsPub.Publish("cdn.snapshot", `{"event":"snapshot"}`); err != nil {
		// The mutation already succeeded. Edges also reconcile periodically, so
		// report the notification failure without returning a misleading API error.
		log.Printf("failed to publish CDN snapshot notification: %v", err)
	}
}
