package client

import (
	"encoding/json"
	"fmt"
	"io"
	"mid/internal/domain"
	"net/http"
	"time"
)

type ControlPanelClientInterface interface {
	GetCDNs() ([]domain.CDN, error)
}

type controlPanelClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewControlPanelClient(baseURL string) ControlPanelClientInterface {
	return &controlPanelClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *controlPanelClient) GetCDNs() ([]domain.CDN, error) {
	url := fmt.Sprintf("%s/cdns", c.baseURL)

	resp, err := c.httpClient.Get(url)
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
