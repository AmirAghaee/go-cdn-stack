package httpserver

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"

	cachehttp "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/httpapi"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewPublic(address string, handler *cachehttp.Handler) *http.Server {
	router := newRouter()
	// Forwarding trust is handled explicitly by the HTTP adapter.
	_ = router.SetTrustedProxies(nil)
	handler.Register(router)
	return &http.Server{Addr: address, Handler: router}
}

func NewInternal(address string) *http.Server {
	router := newRouter()
	// Forwarding trust is handled explicitly by the HTTP adapter.
	_ = router.SetTrustedProxies(nil)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return &http.Server{Addr: address, Handler: router}
}

func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), recovery())
	return router
}

// recovery lets net/http handle ErrAbortHandler so a partially streamed
// response is terminated without a successful HTTP trailer. Other panics retain
// the existing recovery behavior and become internal server errors.
func recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(http.ErrAbortHandler)
			}
			log.Printf("operation=recover_http_panic panic=%q stack=%q", recovered, debug.Stack())
			c.AbortWithStatus(http.StatusInternalServerError)
		}()
		c.Next()
	}
}
