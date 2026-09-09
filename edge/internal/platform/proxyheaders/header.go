// Package proxyheaders contains HTTP adapter header sanitation shared by both hops.
package proxyheaders

import (
	"net/http"
	"strings"
)

// EndToEnd clones headers, removing connection-specific fields, including every
// field nominated by Connection. It also handles noncanonical cached map keys.
func EndToEnd(source map[string][]string) http.Header {
	blocked := map[string]bool{}
	for _, name := range []string{"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade"} {
		blocked[strings.ToLower(name)] = true
	}
	for name, values := range source {
		if strings.EqualFold(name, "Connection") {
			for _, value := range values {
				for _, token := range strings.Split(value, ",") {
					blocked[strings.ToLower(strings.TrimSpace(token))] = true
				}
			}
		}
	}
	result := make(http.Header, len(source))
	for name, values := range source {
		if !blocked[strings.ToLower(name)] {
			for _, value := range values {
				result.Add(name, value)
			}
		}
	}
	return result
}

// RemoveForwarding drops alternate client-controlled representations so origins
// receive only the forwarding metadata explicitly constructed by the edge.
func RemoveForwarding(header http.Header) {
	for name := range header {
		lower := strings.ToLower(name)
		if lower == "forwarded" || lower == "x-real-ip" || strings.HasPrefix(lower, "x-forwarded-") {
			delete(header, name)
		}
	}
}
