package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"sync"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/proxyheaders"
	"github.com/gin-gonic/gin"
)

const streamBufferSize = 32 * 1024

var streamBufferPool = sync.Pool{
	New: func() any { return make([]byte, streamBufferSize) },
}

type Service interface {
	Handle(context.Context, cache.Request) cache.Response
}

type Handler struct {
	service        Service
	trustedProxies []netip.Prefix
}

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *gin.Engine) {
	router.Any("/*path", h.handle)
}

func (h *Handler) handle(c *gin.Context) {
	clientIP, forwardedFor, scheme := h.forwarding(c.Request)
	header := c.Request.Header.Clone()
	proxyheaders.RemoveForwarding(header)
	response := h.service.Handle(c.Request.Context(), cache.Request{
		Method: c.Request.Method, Host: c.Request.Host, URI: c.Request.URL.RequestURI(),
		Header: header, Body: c.Request.Body, ClientIP: clientIP,
		ForwardedFor: forwardedFor, Scheme: scheme,
	})
	for key, values := range proxyheaders.EndToEnd(response.Header) {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	defer response.Body.Close()
	c.Status(response.StatusCode)
	c.Writer.Flush()
	buffer := streamBufferPool.Get().([]byte)
	defer streamBufferPool.Put(buffer)
	if _, err := io.CopyBuffer(c.Writer, response.Body, buffer); err != nil {
		_ = c.Error(fmt.Errorf("stream edge response: %w", err))
	}
}
