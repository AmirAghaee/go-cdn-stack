package http

import (
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/handler/middleware"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/service"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
	"github.com/AmirAghaee/go-cdn-stack/pkg/messaging"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	g *gin.Engine,
	cdnSvc service.CdnServiceInterface,
	userSvc service.UserServiceInterface,
	natsPub messaging.MessageBrokerInterface,
	jwtManager *jwt.Manager,
) {
	// Create JWT middleware
	authMiddleware := middleware.JWTAuth(jwtManager)

	// Protected routes group
	protected := g.Group("/api")
	protected.Use(authMiddleware)

	// Register handlers
	NewUserHandler(userSvc).Register(g, protected)
	NewCdnHandler(cdnSvc).Register(protected)
	NewSnapshotHandler(natsPub).Register(protected)
}
