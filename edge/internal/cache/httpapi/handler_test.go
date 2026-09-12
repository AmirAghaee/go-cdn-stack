package httpapi

import (
	"context"
	"errors"
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

func TestHandlerAbortsResponseWhenBodyStreamingFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := &trackedBody{Reader: io.MultiReader(strings.NewReader("partial"), failingReader{})}
	service := &fakeService{response: cache.Response{
		StatusCode: http.StatusOK,
		Header:     map[string][]string{"Content-Type": {"application/octet-stream"}},
		Body:       body,
	}}
	router := gin.New()
	New(service).Register(router)

	defer func() {
		recovered := recover()
		if !errors.Is(asError(recovered), http.ErrAbortHandler) {
			t.Fatalf("panic = %v, want http.ErrAbortHandler", recovered)
		}
		if !body.closed {
			t.Fatal("response body was not closed")
		}
	}()

	request := httptest.NewRequest(http.MethodGet, "http://cdn.example/asset", nil)
	router.ServeHTTP(httptest.NewRecorder(), request)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("upstream interrupted") }

func asError(value any) error {
	err, _ := value.(error)
	return err
}

func TestForwardingTrustBoundary(t *testing.T) {
	tests := []struct {
		name, peer, chain, proto, connection, wantIP, wantChain, wantProto string
		trusted                                                            []string
	}{
		{name: "default ignores spoofing", peer: "192.0.2.1:1234", chain: "198.51.100.1", proto: "https", wantIP: "192.0.2.1", wantChain: "192.0.2.1", wantProto: "http"},
		{name: "untrusted peer", peer: "192.0.2.1:1234", chain: "198.51.100.1", proto: "https", trusted: []string{"10.0.0.0/8"}, wantIP: "192.0.2.1", wantChain: "192.0.2.1", wantProto: "http"},
		{name: "trusted suffix", peer: "10.0.0.2:1234", chain: "203.0.113.99, 198.51.100.1, 10.0.0.1", proto: "https", trusted: []string{"10.0.0.0/8"}, wantIP: "198.51.100.1", wantChain: "198.51.100.1, 10.0.0.1, 10.0.0.2", wantProto: "https"},
		{name: "malformed chain", peer: "10.0.0.2:1234", chain: "bad, 198.51.100.1", proto: "https,http", trusted: []string{"10.0.0.0/8"}, wantIP: "10.0.0.2", wantChain: "10.0.0.2", wantProto: "http"},
		{name: "IPv6", peer: "[2001:db8::2]:1234", chain: "2001:db8:1::1", trusted: []string{"2001:db8::/64"}, wantIP: "2001:db8:1::1", wantChain: "2001:db8:1::1, 2001:db8::2", wantProto: "http"},
		{name: "connection nominations", peer: "10.0.0.2:1234", chain: "198.51.100.1", proto: "https", connection: "X-Forwarded-For, X-Forwarded-Proto", trusted: []string{"10.0.0.0/8"}, wantIP: "10.0.0.2", wantChain: "10.0.0.2", wantProto: "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeService{response: cache.Response{StatusCode: 200, Header: map[string][]string{"connection": {"X-Secret"}, "X-Secret": {"cached secret"}, "X-Origin": {"preserved"}}, Body: io.NopCloser(strings.NewReader("body"))}}
			handler, err := NewWithTrustedProxies(service, tt.trusted)
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			handler.Register(router)
			request := httptest.NewRequest("GET", "http://cdn.example/asset", nil)
			request.RemoteAddr = tt.peer
			request.Header.Set("X-Forwarded-For", tt.chain)
			request.Header.Set("X-Forwarded-Proto", tt.proto)
			request.Header.Set("Connection", tt.connection)
			request.Header.Set("Forwarded", "for=spoofed")
			request.Header.Set("X-Real-IP", "spoofed")
			request.Header.Set("X-Forwarded-Host", "spoofed")
			request.Header.Set("X-Forwarded-Port", "9999")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			got := service.request
			if got.ClientIP != tt.wantIP || got.ForwardedFor != tt.wantChain || got.Scheme != tt.wantProto {
				t.Fatalf("forwarding = %q %q %q", got.ClientIP, got.ForwardedFor, got.Scheme)
			}
			for name := range got.Header {
				if name == "Forwarded" || name == "X-Real-Ip" || strings.HasPrefix(name, "X-Forwarded-") {
					t.Fatalf("untrusted header retained: %s", name)
				}
			}
			if response.Header().Get("X-Secret") != "" || response.Header().Get("Connection") != "" || response.Header().Get("X-Origin") != "preserved" {
				t.Fatalf("response headers = %v", response.Header())
			}
		})
	}
}

func TestRejectInvalidTrustedProxyCIDR(t *testing.T) {
	for _, cidr := range []string{"bad", "10.0.0.1", ""} {
		if _, err := NewWithTrustedProxies(&fakeService{}, []string{cidr}); err == nil {
			t.Fatalf("accepted %q", cidr)
		}
	}
}
