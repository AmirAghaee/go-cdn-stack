package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
)

type ControlPanelClientInterface interface {
	GetCDNs(ctx context.Context) ([]domain.CDN, error)
}

type controlPanelClient struct {
	baseURL    string
	httpClient *http.Client
	jwtManager *jwt.Manager
}

func NewControlPanelClient(baseURL, jwtSecret string) ControlPanelClientInterface {
	return &controlPanelClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtManager: jwt.NewJWTManager(jwtSecret, 24*time.Hour),
	}
}

func (c *controlPanelClient) GetCDNs(ctx context.Context) ([]domain.CDN, error) {
	token, err := c.jwtManager.Generate("edge", "edge@cdn.internal")
	if err != nil {
		return nil, fmt.Errorf("generate service token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/cdns", nil)
	if err != nil {
		return nil, fmt.Errorf("create control-panel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request control-panel snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("control panel returned status %d", resp.StatusCode)
	}

	var cdns []domain.CDN
	if err := json.NewDecoder(resp.Body).Decode(&cdns); err != nil {
		return nil, fmt.Errorf("decode control-panel snapshot: %w", err)
	}
	return cdns, nil
}
