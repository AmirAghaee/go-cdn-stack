package http

import (
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/messaging"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cdnSvc service.CdnServiceInterface, userSvc service.UserServiceInterface, natsPub messaging.MessageBrokerInterface) {
	NewCdnHandler(cdnSvc).Register(r)
	NewUserHandler(userSvc).Register(r)
	NewSnapshotHandler(natsPub).Register(r)
}
