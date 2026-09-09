package httpapi

import (
	"context"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/gin-gonic/gin"
)

type Service interface {
	Handle(context.Context, cache.Request) cache.Response
}

type Handler struct {
	service Service
}

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *gin.Engine) {
	router.Any("/*path", h.handle)
}

func (h *Handler) handle(c *gin.Context) {
	response := h.service.Handle(c.Request.Context(), cache.Request{
		Method: c.Request.Method, Host: c.Request.Host, URI: c.Request.URL.RequestURI(),
		Header: c.Request.Header.Clone(), Body: c.Request.Body, ClientIP: c.ClientIP(),
	})
	for key, values := range response.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Data(response.StatusCode, c.Writer.Header().Get("Content-Type"), response.Body)
}
