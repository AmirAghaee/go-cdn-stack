package service

import (
	"fmt"
	"log"
	"mid/internal/client"
)

type CdnSnapshotServiceInterface interface {
	ProcessSnapshot() error
}

type cdnSnapshotService struct {
	controlPanelClient client.ControlPanelClientInterface
}

func NewCdnSnapshotService(controlPanelClient client.ControlPanelClientInterface) CdnSnapshotServiceInterface {
	return &cdnSnapshotService{
		controlPanelClient: controlPanelClient,
	}
}

func (s *cdnSnapshotService) ProcessSnapshot() error {
	fmt.Println("📦 Processing CDN snapshot...")

	// Get CDN data from control panel
	cdns, err := s.controlPanelClient.GetCDNs()
	if err != nil {
		return fmt.Errorf("failed to get CDNs from control panel: %w", err)
	}

	log.Printf("🔗 Retrieved %d CDNs from control panel", len(cdns))

	// Process each CDN
	// cdns list
	return nil
}
