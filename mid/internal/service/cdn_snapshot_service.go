package service

import "fmt"

type CdnSnapshotServiceInterface interface {
	ProcessSnapshot(msg string) error
}

type cdnSnapshotService struct{}

func NewCdnSnapshotService() CdnSnapshotServiceInterface {
	return &cdnSnapshotService{}
}

func (s *cdnSnapshotService) ProcessSnapshot(msg string) error {
	// business logic goes here
	fmt.Println("📦 Processing CDN snapshot:", msg)
	return nil
}
