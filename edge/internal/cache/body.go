package cache

import (
	"io"
	"sync"
)

type meteredBody struct {
	body     io.ReadCloser
	count    int64
	onFinish func(int64, error)
	once     sync.Once
}

func newMeteredBody(body io.ReadCloser, onFinish func(int64, error)) io.ReadCloser {
	return &meteredBody{body: body, onFinish: onFinish}
}

func (b *meteredBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	b.count += int64(count)
	if err != nil {
		b.finish(err)
	}
	return count, err
}

func (b *meteredBody) Close() error {
	err := b.body.Close()
	b.finish(err)
	return err
}

func (b *meteredBody) finish(err error) {
	b.once.Do(func() { b.onFinish(b.count, err) })
}

type cacheFillBody struct {
	body          io.ReadCloser
	pending       PendingEntry
	maxObjectSize int64
	written       int64
	active        bool
	finished      bool
	onError       func()
}

func newCacheFillBody(body io.ReadCloser, pending PendingEntry, maxObjectSize int64, onError func()) io.ReadCloser {
	return &cacheFillBody{
		body: body, pending: pending, maxObjectSize: maxObjectSize,
		active: true, onError: onError,
	}
}

func (b *cacheFillBody) Read(buffer []byte) (int, error) {
	count, readErr := b.body.Read(buffer)
	if count > 0 && b.active {
		if b.maxObjectSize > 0 && b.written+int64(count) > b.maxObjectSize {
			b.abort(false)
		} else {
			written, writeErr := b.pending.Write(buffer[:count])
			b.written += int64(written)
			if writeErr != nil || written != count {
				b.abort(true)
			}
		}
	}

	if readErr == io.EOF {
		b.finished = true
		if b.active {
			b.active = false
			if err := b.pending.Commit(); err != nil {
				b.onError()
			}
		}
	} else if readErr != nil {
		b.abort(false)
	}
	return count, readErr
}

func (b *cacheFillBody) Close() error {
	if !b.finished {
		b.abort(false)
	}
	return b.body.Close()
}

func (b *cacheFillBody) abort(recordError bool) {
	if !b.active {
		return
	}
	b.active = false
	if err := b.pending.Abort(); err != nil || recordError {
		b.onError()
	}
}

type flightBody struct {
	body     io.ReadCloser
	onFinish func()
	once     sync.Once
}

func newFlightBody(body io.ReadCloser, onFinish func()) io.ReadCloser {
	return &flightBody{body: body, onFinish: onFinish}
}

func (b *flightBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	if err != nil {
		b.once.Do(b.onFinish)
	}
	return count, err
}

func (b *flightBody) Close() error {
	err := b.body.Close()
	b.once.Do(b.onFinish)
	return err
}
