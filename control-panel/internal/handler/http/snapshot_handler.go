package http

import (
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"
	"github.com/gin-gonic/gin"
)

type SnapshotHandler struct {
	natsPub messaging.MessageBrokerInterface
}

func NewSnapshotHandler(natsPub messaging.MessageBrokerInterface) *SnapshotHandler {
	return &SnapshotHandler{natsPub: natsPub}
}

func (h *SnapshotHandler) Register(r *gin.Engine) {
	r.POST("/snapshot", h.snapshot)
}

func (h *SnapshotHandler) snapshot(c *gin.Context) {
	if err := h.natsPub.Publish("cdn.snapshot", `{"event":"snapshot"}`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish snapshot"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "snapshot published"})
}
