package httpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	cachehttp "github.com/AmirAghaee/go-cdn-stack/edge/internal/cache/httpapi"
	"github.com/gin-gonic/gin"
)

type interruptedService struct{}

func (interruptedService) Handle(context.Context, cache.Request) cache.Response {
	return cache.Response{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"application/octet-stream"}},
		Body:       io.NopCloser(io.MultiReader(strings.NewReader("partial"), interruptedReader{})),
	}
}

type interruptedReader struct{}

func (interruptedReader) Read([]byte) (int, error) { return 0, errors.New("upstream interrupted") }

func TestPublicServerTerminatesInterruptedResponse(t *testing.T) {
	server := NewPublic("", cachehttp.New(interruptedService{}))
	request := httptest.NewRequest(http.MethodGet, "http://cdn.example/asset", nil)
	response := httptest.NewRecorder()

	defer func() {
		recovered, ok := recover().(error)
		if !ok || !errors.Is(recovered, http.ErrAbortHandler) {
			t.Fatalf("panic = %v, want http.ErrAbortHandler", recovered)
		}
		if body := response.Body.String(); body != "partial" {
			t.Fatalf("body = %q, want partial upstream body", body)
		}
	}()

	server.Handler.ServeHTTP(response, request)
}

func TestPublicServerRecoversOtherPanics(t *testing.T) {
	router := newRouter()
	router.GET("/panic", func(*gin.Context) { panic("boom") })
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://edge.example/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
