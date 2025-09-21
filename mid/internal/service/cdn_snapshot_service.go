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
	cdnCacheRepository repository.CdnCacheRepositoryInterface
}

func NewCdnSnapshotService(controlPanelClient client.ControlPanelClientInterface, cache repository.CdnCacheRepositoryInterface) CdnSnapshotServiceInterface {
	return &cdnSnapshotService{
		controlPanelClient: controlPanelClient,
		cdnCacheRepository: cache,
	}
}

func (s *cdnSnapshotService) ProcessSnapshot() error {
	fmt.Println("📦 Processing CDN snapshot...")

	// Get CDN data from control panel
	cdns, err := s.controlPanelClient.GetCDNs()
	if err != nil {
		return fmt.Errorf("failed to get CDNs from control panel: %w", err)
	}

	s.cdnCacheRepository.Set(cdns)

	log.Printf("🔗 Retrieved %d CDNs from control panel", len(cdns))

	return nil
}
