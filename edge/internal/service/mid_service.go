package service

import (
	"log"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/client"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
)

type MidServiceInterface interface {
	StartSubmitHeartbeat()
}

type midService struct {
	midClient     client.MidClientInterface
	config        *config.Config
	cdnRepository repository.CdnRepositoryInterface
	service       string
	instance      string
	version       string
}

func NewMidService(
	midClient client.MidClientInterface,
	cdnRepo repository.CdnRepositoryInterface,
	config *config.Config,
	service, instance, version string,
) MidServiceInterface {
	return &midService{
		midClient:     midClient,
		config:        config,
		cdnRepository: cdnRepo,
		service:       service,
		instance:      instance,
		version:       version,
	}
}

func (s *midService) StartSubmitHeartbeat() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			edge := domain.Edge{
				Service:   s.service,
				Instance:  s.instance,
				Status:    "ok",
				Timestamp: time.Now().UTC(),
				Version:   s.version,
			}

			cdnListVersion, err := s.midClient.Submit(edge)
			if err != nil {
				log.Printf("failed to submit heartbeat: %v\n", err)
				return
			}

			if cdnListVersion != s.cdnRepository.GetVersion() {
				cdns, err := s.midClient.GetCdns()
				if err != nil {
					log.Printf("failed to get cdn list: %s\n", err)
				}
				s.cdnRepository.Set(cdns, cdnListVersion)
				log.Print(cdns)
			}

		}
	}()
}
