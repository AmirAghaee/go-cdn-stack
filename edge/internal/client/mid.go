package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
)

type MidClientInterface interface {
	Submit(edge domain.Edge) (string, error)
	GetCdns() ([]domain.CDN, error)
}

type midClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMidClient(baseURL string) MidClientInterface {
	return &midClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *midClient) Submit(edge domain.Edge) (string, error) {
	url := fmt.Sprintf("http://%s/edge/submit", c.baseURL)

	body, err := json.Marshal(edge)
	if err != nil {
		return "", fmt.Errorf("failed to marshal edge: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to make request to mid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mid service returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var response struct {
		Status         string `json:"status"`
		Instance       string `json:"instance"`
		CdnListVersion string `json:"cdn_list_version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode mid response: %w", err)
	}

	return response.CdnListVersion, nil
}

func (c *midClient) GetCdns() ([]domain.CDN, error) {
	url := fmt.Sprintf("http://%s/edge/cdns", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cdns from mid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mid service returned status %d", resp.StatusCode)
	}

	var cdns []domain.CDN
	if err := json.NewDecoder(resp.Body).Decode(&cdns); err != nil {
		return nil, fmt.Errorf("failed to decode cdns response: %w", err)
	}

	return cdns, nil
}
