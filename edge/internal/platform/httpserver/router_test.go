package httpserver

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	server := NewPublic("", cachehttp.New(interruptedService{}), testLimits())
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
	router := newRouter(1)
	router.GET("/panic", func(*gin.Context) { panic("boom") })
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://edge.example/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestServerAppliesConnectionLimits(t *testing.T) {
	limits := testLimits()
	server := NewInternal("127.0.0.1:8090", limits)

	if server.ReadHeaderTimeout != limits.ReadHeaderTimeout ||
		server.IdleTimeout != limits.IdleTimeout ||
		server.MaxHeaderBytes != limits.MaxHeaderBytes || server.ConnState == nil {
		t.Fatalf("server limits = header timeout %v, idle timeout %v, max header %d", server.ReadHeaderTimeout, server.IdleTimeout, server.MaxHeaderBytes)
	}
	if server.ReadTimeout != 0 || server.WriteTimeout != 0 {
		t.Fatalf("streaming deadlines = read %v, write %v; want zero", server.ReadTimeout, server.WriteTimeout)
	}
}

func TestConnectionLimitClosesExcessConnection(t *testing.T) {
	connectionState := limitConnections(1)
	firstServer, firstClient := net.Pipe()
	defer firstServer.Close()
	defer firstClient.Close()
	connectionState(firstServer, http.StateNew)

	secondServer, secondClient := net.Pipe()
	defer secondClient.Close()
	connectionState(secondServer, http.StateNew)
	_ = secondClient.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := secondClient.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("excess connection read error = %v, want EOF", err)
	}

	connectionState(firstServer, http.StateClosed)
}

func TestRouterRejectsRequestsBeyondConcurrencyLimit(t *testing.T) {
	router := newRouter(1)
	entered := make(chan struct{})
	release := make(chan struct{})
	router.GET("/work", func(c *gin.Context) {
		close(entered)
		<-release
		c.Status(http.StatusNoContent)
	})

	firstResponse := httptest.NewRecorder()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		router.ServeHTTP(firstResponse, httptest.NewRequest(http.MethodGet, "http://edge.example/work", nil))
	}()
	<-entered

	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, httptest.NewRequest(http.MethodGet, "http://edge.example/work", nil))
	if secondResponse.Code != http.StatusServiceUnavailable || secondResponse.Header().Get("Retry-After") != "1" {
		t.Fatalf("overload response = status %d headers %v", secondResponse.Code, secondResponse.Header())
	}

	close(release)
	<-firstDone
	if firstResponse.Code != http.StatusNoContent {
		t.Fatalf("first response status = %d", firstResponse.Code)
	}
}

func testLimits() Limits {
	return Limits{
		ReadHeaderTimeout: time.Second, IdleTimeout: 2 * time.Second,
		MaxHeaderBytes: 1024, MaxConnections: 4, MaxConcurrentRequests: 2,
	}
}
