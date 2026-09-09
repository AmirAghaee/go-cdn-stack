package httpserver

import (
	cdnhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/httpapi"
	identityhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/httpapi"
	"github.com/gin-gonic/gin"
)

func New(identityHandler *identityhttp.Handler, cdnHandler *cdnhttp.Handler, auth gin.HandlerFunc) *gin.Engine {
	router := gin.Default()
	identityHandler.RegisterPublic(router)

	protected := router.Group("/api")
	protected.Use(auth)
	identityHandler.RegisterProtected(protected)
	cdnHandler.Register(protected)
	return router
}
