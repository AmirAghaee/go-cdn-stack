package service

import (
	"fmt"
	"log"
	"mid/internal/client"
	"mid/internal/domain"
)

type CdnSnapshotServiceInterface interface {
	ProcessSnapshot(msg string) error
}

type cdnSnapshotService struct {
	controlPanelClient client.ControlPanelClientInterface
}

func NewCdnSnapshotService(controlPanelClient client.ControlPanelClientInterface) CdnSnapshotServiceInterface {
	return &cdnSnapshotService{
		controlPanelClient: controlPanelClient,
	}
}

func (s *cdnSnapshotService) ProcessSnapshot(msg string) error {
	fmt.Println("📦 Processing CDN snapshot:", msg)

	// Get CDN data from control panel
	cdns, err := s.controlPanelClient.GetCDNs()
	if err != nil {
		return fmt.Errorf("failed to get CDNs from control panel: %w", err)
	}

	log.Printf("🔗 Retrieved %d CDNs from control panel", len(cdns))

	// Process each CDN
	for _, cdn := range cdns {
		if err := s.processCDN(cdn, msg); err != nil {
			log.Printf("❌ Error processing CDN %s: %v", cdn.ID, err)
			// Continue processing other CDNs even if one fails
		}
	}

	return nil
}

func (s *cdnSnapshotService) processCDN(cdn domain.CDN, snapshotMsg string) error {
	// Your business logic for processing each CDN goes here
	fmt.Printf("🌐 Processing CDN: ID=%s, Domain=%s, Origin=%s", cdn.ID, cdn.Domain, cdn.Origin)

	// Example: you might want to create snapshots, check health, update status, etc.
	// This is where you implement your specific business logic

	return nil
}
