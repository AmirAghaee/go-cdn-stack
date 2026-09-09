package httpapi

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/platform/proxyheaders"
)

// NewWithTrustedProxies accepts only explicit CIDRs; an empty list trusts no peer.
func NewWithTrustedProxies(service Service, cidrs []string) (*Handler, error) {
	h := New(service)
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy CIDR %q: %w", cidr, err)
		}
		h.trustedProxies = append(h.trustedProxies, prefix)
	}
	return h, nil
}

func (h *Handler) trusted(ip netip.Addr) bool {
	for _, prefix := range h.trustedProxies {
		if prefix.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

func (h *Handler) forwarding(r *http.Request) (clientIP, forwardedFor, scheme string) {
	scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", "", scheme
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "", "", scheme
	}
	peer = peer.Unmap()
	clientIP, forwardedFor = peer.String(), peer.String()
	if !h.trusted(peer) {
		return
	}
	header := proxyheaders.EndToEnd(r.Header)
	if values := header.Values("X-Forwarded-Proto"); len(values) == 1 && (values[0] == "http" || values[0] == "https") {
		scheme = values[0]
	}
	values := header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return
	}
	parts := strings.Split(strings.Join(values, ","), ",")
	chain := make([]string, 0, len(parts)+1)
	trusted := make([]bool, 0, len(parts))
	for _, part := range parts {
		ip, parseErr := netip.ParseAddr(strings.TrimSpace(part))
		if parseErr != nil {
			return
		}
		chain = append(chain, ip.Unmap().String())
		trusted = append(trusted, h.trusted(ip))
	}
	// Discard any prefix supplied by the first untrusted hop, retaining that hop
	// and the trusted suffix. The socket peer is always appended, never inferred.
	start := len(chain) - 1
	for start > 0 && trusted[start] {
		start--
	}
	clientIP = chain[start]
	forwardedFor = strings.Join(append(chain[start:], peer.String()), ", ")
	return
}
