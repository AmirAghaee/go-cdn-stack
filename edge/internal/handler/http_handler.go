package handler

import (
	"edge/internal/service"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	edgeService service.EdgeService
}

func NewHTTPHandler(edgeService service.EdgeService) *HTTPHandler {
	return &HTTPHandler{
		edgeService: edgeService,
	}
}

func (h *HTTPHandler) HandleCacheRequest(c *gin.Context) {
	h.edgeService.CacheRequest(c)
}

func (h *HTTPHandler) HandleProxyRequest(c *gin.Context) {
	h.edgeService.ProxyRequest(c)
}
