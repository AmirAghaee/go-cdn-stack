package service

import (
	"fmt"
	"log"
	"mid/internal/client"
	"mid/internal/repository"
)

type CdnSnapshotServiceInterface interface {
	ProcessSnapshot() error
}

type cdnSnapshotService struct {
	controlPanelClient client.ControlPanelClientInterface
	cdnRepository      repository.CdnRepositoryInterface
}

func NewCdnSnapshotService(controlPanelClient client.ControlPanelClientInterface, cache repository.CdnRepositoryInterface) CdnSnapshotServiceInterface {
	return &cdnSnapshotService{
		controlPanelClient: controlPanelClient,
		cdnRepository:      cache,
	}
}

func (s *cdnSnapshotService) ProcessSnapshot() error {
	fmt.Println("📦 Processing CDN snapshot...")

	// Get CDN data from control panel
	cdns, err := s.controlPanelClient.GetCDNs()
	if err != nil {
		return fmt.Errorf("failed to get CDNs from control panel: %w", err)
	}

	s.cdnRepository.Set(cdns)

	log.Printf("🔗 Retrieved %d CDNs from control panel", len(cdns))

	return nil
}
