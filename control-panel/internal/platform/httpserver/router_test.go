package httpserver

import (
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

func TestRegisterRequiresJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(
		identityhttp.New(identity.NewService(nil, nil)),
		cdnhttp.New(cdn.NewService(nil, nil)),
		identityhttp.Auth(rejectingTokenVerifier{}),
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
