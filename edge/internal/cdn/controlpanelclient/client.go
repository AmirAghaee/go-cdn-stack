package controlpanelclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	jwtManager *jwt.Manager
}

type responseDTO struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Origin   string `json:"origin"`
	IsActive bool   `json:"is_active"`
	CacheTTL uint   `json:"cache_ttl"`
}

func New(baseURL, jwtSecret string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		jwtManager: jwt.NewJWTManager(jwtSecret, 24*time.Hour),
	}
}

func (c *Client) Fetch(ctx context.Context) ([]cdn.CDN, error) {
	token, err := c.jwtManager.Generate("edge", "edge@cdn.internal")
	if err != nil {
		return nil, fmt.Errorf("generate service token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/cdns", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var payload []responseDTO
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	items := make([]cdn.CDN, 0, len(payload))
	for _, dto := range payload {
		item, err := cdn.New(dto.ID, dto.Domain, dto.Origin, dto.IsActive, dto.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("validate CDN %q: %w", dto.Domain, err)
		}
		items = append(items, item)
	}
	return items, nil
}
