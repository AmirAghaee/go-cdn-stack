package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	request  cache.Request
	response cache.Response
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func (f *fakeService) Handle(_ context.Context, request cache.Request) cache.Response {
	f.request = request
	return f.response
}

func TestHandlerMapsHTTPRequestAndApplicationResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := &trackedBody{Reader: strings.NewReader("image")}
	service := &fakeService{response: cache.Response{
		StatusCode: http.StatusCreated,
		Header:     map[string][]string{"Content-Type": {"image/png"}, "X-Origin": {"one", "two"}},
		Body:       body,
	}}
	router := gin.New()
	New(service).Register(router)

	request := httptest.NewRequest(http.MethodGet, "http://cdn.example/asset?id=7", nil)
	request.Host = "cdn.example"
	request.Header.Set("X-Test", "preserved")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if service.request.Host != "cdn.example" || service.request.URI != "/asset?id=7" {
		t.Fatalf("mapped request host=%q URI=%q", service.request.Host, service.request.URI)
	}
	if response.Code != http.StatusCreated || response.Body.String() != "image" {
		t.Fatalf("response status=%d body=%q", response.Code, response.Body.String())
	}
	if values := response.Header().Values("X-Origin"); len(values) != 2 {
		t.Fatalf("X-Origin values = %v", values)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}
