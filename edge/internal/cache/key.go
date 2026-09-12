package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
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

// DomainFromKey returns the normalized CDN domain encoded in a current cache
// key. It rejects incomplete and non-canonical keys so maintenance operations
// cannot purge an entry based on an ambiguous substring match.
func DomainFromKey(key string) (string, bool) {
	remainder, ok := strings.CutPrefix(key, cacheKeyVersion+"|configuration=")
	if !ok || len(remainder) < sha256.Size*2+len("|host=") {
		return "", false
	}
	configuration := remainder[:sha256.Size*2]
	if _, err := hex.DecodeString(configuration); err != nil {
		return "", false
	}
	remainder = remainder[sha256.Size*2:]
	if !strings.HasPrefix(remainder, "|host=") {
		return "", false
	}

	domain, remainder, ok := consumeKeyValue(remainder[1:], "host=")
	if !ok || domain == "" || cdn.NormalizeDomain(domain) != domain || !strings.HasPrefix(remainder, "|uri=") {
		return "", false
	}
	_, remainder, ok = consumeKeyValue(remainder[1:], "uri=")
	if !ok || !strings.HasPrefix(remainder, "|accept-encoding=") {
		return "", false
	}
	_, remainder, ok = consumeKeyValue(remainder[1:], "accept-encoding=")
	if !ok || remainder != "" {
		return "", false
	}
	return domain, true
}

func consumeKeyValue(value, prefix string) (string, string, bool) {
	value, ok := strings.CutPrefix(value, prefix)
	if !ok {
		return "", "", false
	}
	separator := strings.IndexByte(value, ':')
	if separator <= 0 {
		return "", "", false
	}
	lengthText := value[:separator]
	length, err := strconv.Atoi(lengthText)
	if err != nil || length < 0 || strconv.Itoa(length) != lengthText {
		return "", "", false
	}
	value = value[separator+1:]
	if length > len(value) {
		return "", "", false
	}
	return value[:length], value[length:], true
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
