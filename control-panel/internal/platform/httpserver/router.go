package httpserver

import (
	cdnhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/httpapi"
	healthhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth/httpapi"
	identityhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/httpapi"
	"github.com/gin-gonic/gin"
)

func New(identityHandler *identityhttp.Handler, cdnHandler *cdnhttp.Handler, adminAuth, edgeAuth gin.HandlerFunc, healthHandlers ...*healthhttp.Handler) *gin.Engine {
	router := gin.Default()
	identityHandler.RegisterPublic(router)

	protected := router.Group("/api")
	protected.Use(adminAuth)
	identityHandler.RegisterProtected(protected)
	cdnHandler.Register(protected)
	for _, healthHandler := range healthHandlers {
		healthHandler.Register(protected)
	}

	edge := router.Group("/edge/v1")
	edge.Use(edgeAuth)
	cdnHandler.RegisterSnapshot(edge)
	return router
}
