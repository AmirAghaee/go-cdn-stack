package http

import (
	"control-panel/internal/messaging"
	"control-panel/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cdnSvc service.CdnServiceInterface, userSvc service.UserServiceInterface, natsPub messaging.MessageBrokerInterface) {
	NewCdnHandler(cdnSvc).Register(r)
	NewUserHandler(userSvc).Register(r)
	NewSnapshotHandler(natsPub).Register(r)
}
