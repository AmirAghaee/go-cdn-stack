package cache

import (
	"io"
	"time"
)

type Request struct {
	Method   string
	Host     string
	URI      string
	Header   map[string][]string
	Body     io.Reader
	ClientIP string
	// ForwardedFor and Scheme are established by the inbound HTTP trust boundary.
	ForwardedFor string
	Scheme       string
}

// Response carries the selected response stream. The caller must close Body,
// including when it is not read to EOF.
type Response struct {
	StatusCode  int
	Header      map[string][]string
	Body        io.ReadCloser
	CacheStatus string
}

type Entry struct {
	StatusCode int
	Header     map[string][]string
	Body       io.ReadCloser
	ExpiresAt  time.Time
	StoredAt   time.Time
	InitialAge time.Duration
}

type EntryMetadata struct {
	StatusCode int
	Header     map[string][]string
	ExpiresAt  time.Time
	// StoredAt is the origin-header receipt time; InitialAge is the corrected
	// response age at that instant.
	StoredAt   time.Time
	InitialAge time.Duration
}

// PendingEntry owns an unpublished cache body. The caller must finish it with
// Commit or Abort. Commit is terminal even when it returns an error; the
// implementation is responsible for cleaning up its pending resources.
type PendingEntry interface {
	io.Writer
	Commit() error
	Abort() error
}

type OriginRequest struct {
	Method   string
	Origin   string
	URI      string
	Host     string
	Header   map[string][]string
	Body     io.Reader
	ClientIP string
	// ForwardedFor and Scheme are established by the inbound HTTP trust boundary.
	ForwardedFor string
	Scheme       string
}

type OriginResponse struct {
	StatusCode    int
	Header        map[string][]string
	Body          io.ReadCloser
	ContentLength int64
}
