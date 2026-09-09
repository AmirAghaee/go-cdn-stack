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
	Body        []byte
	CacheStatus string
}

type Entry struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
	ExpiresAt  time.Time
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
	StatusCode int
	Header     map[string][]string
	Body       []byte
}
