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
	Submit(edge domain.Edge) error
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

func (c *midClient) Submit(edge domain.Edge) error {
	url := fmt.Sprintf("http://%s/edge/submit", c.baseURL)

	body, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("failed to marshal edge: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to make request to mid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mid service returned status %d", resp.StatusCode)
	}

	return nil
}
