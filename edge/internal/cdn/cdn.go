package cdn

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// CDN is a validated edge configuration entry.
type CDN struct {
	id       string
	domain   string
	origin   string
	isActive bool
	cacheTTL uint
}

func New(id, domain, origin string, isActive bool, cacheTTL uint) (CDN, error) {
	domain = NormalizeDomain(domain)
	if domain == "" {
		return CDN{}, fmt.Errorf("CDN domain is required")
	}

	originURL, err := url.Parse(origin)
	if err != nil || (originURL.Scheme != "http" && originURL.Scheme != "https") || originURL.Host == "" {
		return CDN{}, fmt.Errorf("CDN origin must be an HTTP or HTTPS URL")
	}

	return CDN{id: id, domain: domain, origin: strings.TrimRight(origin, "/"), isActive: isActive, cacheTTL: cacheTTL}, nil
}

func NormalizeDomain(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return strings.TrimSuffix(value, ".")
}

func (c CDN) ID() string     { return c.id }
func (c CDN) Domain() string { return c.domain }
func (c CDN) Origin() string { return c.origin }
func (c CDN) IsActive() bool { return c.isActive }
func (c CDN) CacheTTL() uint { return c.cacheTTL }
