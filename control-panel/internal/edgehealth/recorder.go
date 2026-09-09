package edgehealth

import "context"

type Store interface {
	Upsert(context.Context, Status) error
}

type Recorder struct {
	store Store
}

func NewRecorder(store Store) *Recorder {
	return &Recorder{store: store}
}

func (r *Recorder) Record(ctx context.Context, status Status) error {
	return r.store.Upsert(ctx, status)
}
