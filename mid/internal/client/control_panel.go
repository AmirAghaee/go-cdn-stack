package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
)

type ControlPanelClientInterface interface {
	GetCDNs() ([]domain.CDN, error)
}

type controlPanelClient struct {
	baseURL    string
	httpClient *http.Client
	jwtManager *jwt.Manager
}

func NewControlPanelClient(baseURL string, jwtSecretKey string) ControlPanelClientInterface {
	return &controlPanelClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtManager: jwt.NewJWTManager(jwtSecretKey, 24*time.Hour),
	}
}

func (c *controlPanelClient) GetCDNs() ([]domain.CDN, error) {
	url := fmt.Sprintf("%s/api/cdns", c.baseURL)

	token, err := c.jwtManager.Generate("0", "mid01@cdn.lab")
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to control panel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("control panel returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var cdns []domain.CDN
	if err := json.Unmarshal(body, &cdns); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return cdns, nil
}
