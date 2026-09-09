package httpapi

import (
	"log"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *edgehealth.Service
}

func New(service *edgehealth.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(routes *gin.RouterGroup) {
	routes.GET("/health/nodes", h.list)
}

func (h *Handler) list(c *gin.Context) {
	statuses, err := h.service.List(c.Request.Context())
	if err != nil {
		log.Printf("failed to list edge health statuses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list node health"})
		return
	}

	response := make([]nodeResponse, 0, len(statuses))
	for _, status := range statuses {
		response = append(response, newNodeResponse(status))
	}
	c.JSON(http.StatusOK, response)
}
