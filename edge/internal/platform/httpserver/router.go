package httpserver

import (
	"net/http"

	cachehttp "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/httpapi"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewPublic(address string, handler *cachehttp.Handler) *http.Server {
	router := gin.Default()
	handler.Register(router)
	return &http.Server{Addr: address, Handler: router}
}

func NewInternal(address string) *http.Server {
	router := gin.Default()
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return &http.Server{Addr: address, Handler: router}
}
