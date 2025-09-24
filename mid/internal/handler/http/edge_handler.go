package http

import (
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/service"

	"github.com/gin-gonic/gin"
)

type EdgeHandler struct {
	edgeService service.EdgeServiceInterface
}

func NewEdgeHandler(edgeService service.EdgeServiceInterface) *EdgeHandler {
	return &EdgeHandler{
		edgeService: edgeService,
	}
}

func (h *EdgeHandler) Register(r *gin.Engine) {
	r.POST("/edge/submit", h.edgeService.Register)
}
