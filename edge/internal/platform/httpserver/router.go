package httpserver

import (
	"errors"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	cachehttp "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/httpapi"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Limits struct {
	ReadHeaderTimeout     time.Duration
	IdleTimeout           time.Duration
	MaxHeaderBytes        int
	MaxConnections        int
	MaxConcurrentRequests int
}

type Readiness interface {
	Ready() bool
}

func NewPublic(address string, handler *cachehttp.Handler, limits Limits) *http.Server {
	router := newRouter(limits.MaxConcurrentRequests)
	// Forwarding trust is handled explicitly by the HTTP adapter.
	_ = router.SetTrustedProxies(nil)
	handler.Register(router)
	return newServer(address, router, limits)
}

func NewInternal(address string, readiness Readiness, limits Limits) *http.Server {
	router := newRouter(limits.MaxConcurrentRequests)
	// Forwarding trust is handled explicitly by the HTTP adapter.
	_ = router.SetTrustedProxies(nil)
	router.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if !readiness.Ready() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return newServer(address, router, limits)
}

func newServer(address string, handler http.Handler, limits Limits) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: limits.ReadHeaderTimeout,
		IdleTimeout:       limits.IdleTimeout,
		MaxHeaderBytes:    limits.MaxHeaderBytes,
		ConnState:         limitConnections(limits.MaxConnections),
	}
}

func limitConnections(maxConnections int) func(net.Conn, http.ConnState) {
	var mu sync.Mutex
	admitted := make(map[net.Conn]struct{}, maxConnections)
	return func(connection net.Conn, state http.ConnState) {
		mu.Lock()
		switch state {
		case http.StateNew:
			if len(admitted) >= maxConnections {
				mu.Unlock()
				_ = connection.Close()
				return
			}
			admitted[connection] = struct{}{}
		case http.StateHijacked, http.StateClosed:
			delete(admitted, connection)
		}
		mu.Unlock()
	}
}

func newRouter(maxConcurrentRequests int) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), recovery(), limitConcurrency(maxConcurrentRequests))
	return router
}

func limitConcurrency(maxConcurrentRequests int) gin.HandlerFunc {
	active := make(chan struct{}, maxConcurrentRequests)
	return func(c *gin.Context) {
		select {
		case active <- struct{}{}:
			defer func() { <-active }()
			c.Next()
		default:
			c.Header("Retry-After", "1")
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
	}
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
