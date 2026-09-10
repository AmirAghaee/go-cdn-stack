package cache

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

const cacheKeyVersion = "v3"

func cacheKey(item cdn.CDN, uri string, header map[string][]string) string {
	acceptEncoding := normalizeAcceptEncoding(headerValues(header, "Accept-Encoding"))
	configuration := sha256.Sum256([]byte(fmt.Sprintf(
		"id=%d:%s|domain=%d:%s|origin=%d:%s|active=%t|ttl=%d",
		len(item.ID()),
		item.ID(),
		len(item.Domain()),
		item.Domain(),
		len(item.Origin()),
		item.Origin(),
		item.IsActive(),
		item.CacheTTL(),
	)))
	return fmt.Sprintf(
		"%s|configuration=%x|host=%d:%s|uri=%d:%s|accept-encoding=%d:%s",
		cacheKeyVersion,
		configuration,
		len(item.Domain()), item.Domain(),
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
