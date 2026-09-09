package edgehealth

import "context"

type Reader interface {
	List(context.Context) ([]Status, error)
}

type Service struct {
	reader Reader
}

func NewService(reader Reader) *Service {
	return &Service{reader: reader}
}

func (s *Service) List(ctx context.Context) ([]Status, error) {
	return s.reader.List(ctx)
}
