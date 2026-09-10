package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn"
	cdnhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn/httpapi"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	identityhttp "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity/httpapi"
	"github.com/gin-gonic/gin"
)

type rejectingTokenVerifier struct{}

func (rejectingTokenVerifier) Verify(string) (identity.Claims, error) {
	return identity.Claims{}, errors.New("invalid token")
}

type adminOnlyTokenVerifier struct{}

func (adminOnlyTokenVerifier) Verify(token string) (identity.Claims, error) {
	if token != "admin-token" {
		return identity.Claims{}, errors.New("invalid token")
	}
	return identity.Claims{UserID: "admin"}, nil
}

type snapshotStore struct{}

func (snapshotStore) Create(context.Context, *cdn.CDN) error                 { return nil }
func (snapshotStore) List(context.Context) ([]*cdn.CDN, error)               { return []*cdn.CDN{}, nil }
func (snapshotStore) Get(context.Context, string) (*cdn.CDN, error)          { return nil, nil }
func (snapshotStore) Update(context.Context, string, *cdn.CDN) error         { return nil }
func (snapshotStore) Delete(context.Context, string) error                   { return nil }
func (snapshotStore) FindByOrigin(context.Context, string) (*cdn.CDN, error) { return nil, nil }

func TestRegisterRequiresJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(
		identityhttp.New(identity.NewService(nil, nil)),
		cdnhttp.New(cdn.NewService(nil, nil)),
		identityhttp.Auth(rejectingTokenVerifier{}),
		identityhttp.ServiceAuth("edge-secret"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{
		"email":"admin@example.com","password":"secret"
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/register status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestEdgeCredentialIsScopedToSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(
		identityhttp.New(identity.NewService(nil, nil)),
		cdnhttp.New(cdn.NewService(snapshotStore{}, nil)),
		identityhttp.Auth(adminOnlyTokenVerifier{}),
		identityhttp.ServiceAuth("edge-secret"),
	)

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
	}{
		{name: "edge reads snapshot", method: http.MethodGet, path: "/edge/v1/snapshot", token: "edge-secret", wantStatus: http.StatusOK},
		{name: "edge cannot mutate CDN", method: http.MethodDelete, path: "/api/cdns/cdn-1", token: "edge-secret", wantStatus: http.StatusUnauthorized},
		{name: "admin cannot use edge route", method: http.MethodGet, path: "/edge/v1/snapshot", token: "admin-token", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			request.Header.Set("Authorization", "Bearer "+tt.token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.wantStatus)
			}
		})
	}
}
