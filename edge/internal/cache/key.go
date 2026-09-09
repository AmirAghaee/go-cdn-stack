package cache

import (
	"fmt"
	"strings"
)

const cacheKeyVersion = "v2"

func cacheKey(host, uri string, header map[string][]string) string {
	acceptEncoding := normalizeAcceptEncoding(headerValues(header, "Accept-Encoding"))
	return fmt.Sprintf(
		"%s|host=%d:%s|uri=%d:%s|accept-encoding=%d:%s",
		cacheKeyVersion,
		len(host), host,
		len(uri), uri,
		len(acceptEncoding), acceptEncoding,
	)
}

func normalizeAcceptEncoding(values []string) string {
	var encodings []string
	for _, value := range values {
		for _, encoding := range splitHeaderList(value) {
			encoding = strings.ToLower(strings.TrimSpace(encoding))
			if encoding != "" {
				encodings = append(encodings, encoding)
			}
		}
	}
	return strings.Join(encodings, ",")
}
