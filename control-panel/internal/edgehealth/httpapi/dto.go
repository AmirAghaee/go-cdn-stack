package httpapi

import (
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/edgehealth"
)

type nodeResponse struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Instance  string    `json:"instance"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

func newNodeResponse(status edgehealth.Status) nodeResponse {
	return nodeResponse{
		ID:        status.ID,
		Service:   status.Service,
		Instance:  status.Instance,
		Status:    status.Status,
		Timestamp: status.Timestamp,
		Version:   status.Version,
	}
}
