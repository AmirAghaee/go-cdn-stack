package handler

import (
	"cdneto/internal/service"

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

func (h *HTTPHandler) HandleRequest(c *gin.Context) {
	h.edgeService.CacheRequest(c)
}
