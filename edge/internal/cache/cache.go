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
}

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
}

type EntryMetadata struct {
	StatusCode int
	Header     map[string][]string
	ExpiresAt  time.Time
}

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
}

type OriginResponse struct {
	StatusCode    int
	Header        map[string][]string
	Body          io.ReadCloser
	ContentLength int64
}
