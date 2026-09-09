package cache

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
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

func shouldBypassCache(request Request) bool {
	if hasHeader(request.Header, "Authorization") ||
		hasHeader(request.Header, "Cookie") ||
		hasHeader(request.Header, "Range") {
		return true
	}

	cacheControl, valid := parseCacheControl(request.Header)
	if !valid {
		return true
	}
	if _, ok := cacheControl["no-cache"]; ok {
		return true
	}
	if _, ok := cacheControl["no-store"]; ok {
		return true
	}

	for _, value := range headerValues(request.Header, "Pragma") {
		for _, directive := range splitHeaderList(value) {
			if strings.EqualFold(strings.TrimSpace(directive), "no-cache") {
				return true
			}
		}
	}

	return false
}

func cacheExpiry(now time.Time, configuredTTL uint, response Response) (time.Time, bool) {
	if response.StatusCode != http.StatusOK ||
		!isCacheableContentType(firstHeader(response.Header, "Content-Type")) ||
		hasHeader(response.Header, "Set-Cookie") ||
		!hasSupportedVary(response.Header) {
		return time.Time{}, false
	}

	configuredDuration, ok := cacheTTLDuration(configuredTTL)
	if !ok {
		return time.Time{}, false
	}
	configuredExpiry := now.Add(configuredDuration)

	directives, valid := parseCacheControl(response.Header)
	if !valid {
		return time.Time{}, false
	}
	for _, directive := range []string{"private", "no-store", "no-cache"} {
		if _, found := directives[directive]; found {
			return time.Time{}, false
		}
	}

	age, ok := parseAge(response.Header)
	if !ok {
		return time.Time{}, false
	}

	if values, found := directives["s-maxage"]; found {
		return boundedFreshnessExpiry(now, configuredExpiry, values, age)
	}
	if values, found := directives["max-age"]; found {
		return boundedFreshnessExpiry(now, configuredExpiry, values, age)
	}

	expiresValues := headerValues(response.Header, "Expires")
	if len(expiresValues) > 0 {
		if len(expiresValues) != 1 {
			return time.Time{}, false
		}
		expiresAt, err := http.ParseTime(strings.TrimSpace(expiresValues[0]))
		if err != nil {
			return time.Time{}, false
		}

		dateValues := headerValues(response.Header, "Date")
		originExpiry := expiresAt.Add(-age)
		if len(dateValues) > 0 {
			if len(dateValues) != 1 {
				return time.Time{}, false
			}
			responseDate, err := http.ParseTime(strings.TrimSpace(dateValues[0]))
			if err != nil {
				return time.Time{}, false
			}
			if !expiresAt.After(responseDate) {
				return time.Time{}, false
			}
			freshnessLifetime := expiresAt.Sub(responseDate)
			apparentAge := now.Sub(responseDate)
			if apparentAge < 0 {
				apparentAge = 0
			}
			currentAge := maxDuration(apparentAge, age)
			originExpiry = now.Add(freshnessLifetime - currentAge)
		}

		return earlierFutureExpiry(now, configuredExpiry, originExpiry)
	}

	return configuredExpiry, true
}

func cacheTTLDuration(ttl uint) (time.Duration, bool) {
	if ttl == 0 || uint64(ttl) > uint64(math.MaxInt64/int64(time.Second)) {
		return 0, false
	}
	return time.Duration(ttl) * time.Second, true
}

func boundedFreshnessExpiry(now, configuredExpiry time.Time, values []string, age time.Duration) (time.Time, bool) {
	if len(values) != 1 || values[0] == "" {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseUint(values[0], 10, 63)
	if err != nil || seconds > uint64(math.MaxInt64/int64(time.Second)) {
		return time.Time{}, false
	}
	originExpiry := now.Add(time.Duration(seconds)*time.Second - age)
	return earlierFutureExpiry(now, configuredExpiry, originExpiry)
}

func earlierFutureExpiry(now, first, second time.Time) (time.Time, bool) {
	if !second.After(now) {
		return time.Time{}, false
	}
	if second.Before(first) {
		return second, true
	}
	return first, true
}

func parseAge(header map[string][]string) (time.Duration, bool) {
	values := headerValues(header, "Age")
	if len(values) == 0 {
		return 0, true
	}
	if len(values) != 1 {
		return 0, false
	}
	seconds, err := strconv.ParseUint(strings.TrimSpace(values[0]), 10, 63)
	if err != nil || seconds > uint64(math.MaxInt64/int64(time.Second)) {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func hasSupportedVary(header map[string][]string) bool {
	for _, value := range headerValues(header, "Vary") {
		for _, field := range splitHeaderList(value) {
			field = strings.TrimSpace(field)
			if field != "" && !strings.EqualFold(field, "Accept-Encoding") {
				return false
			}
		}
	}
	return true
}

func parseCacheControl(header map[string][]string) (map[string][]string, bool) {
	directives := make(map[string][]string)
	for _, value := range headerValues(header, "Cache-Control") {
		if !hasBalancedQuotes(value) {
			return nil, false
		}
		for _, part := range splitHeaderList(value) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name, directiveValue, hasValue := strings.Cut(part, "=")
			name = strings.ToLower(strings.TrimSpace(name))
			if !isToken(name) {
				return nil, false
			}
			if hasValue {
				directiveValue = strings.TrimSpace(directiveValue)
				if strings.HasPrefix(directiveValue, `"`) {
					unquoted, err := strconv.Unquote(directiveValue)
					if err != nil {
						return nil, false
					}
					directiveValue = unquoted
				} else if !isToken(directiveValue) {
					return nil, false
				}
			} else {
				directiveValue = ""
			}
			directives[name] = append(directives[name], directiveValue)
		}
	}
	return directives, true
}

func hasBalancedQuotes(value string) bool {
	inQuotes := false
	escaped := false
	for _, character := range value {
		switch {
		case escaped:
			escaped = false
		case inQuotes && character == '\\':
			escaped = true
		case character == '"':
			inQuotes = !inQuotes
		}
	}
	return !inQuotes && !escaped
}

func isToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z') &&
			!(character >= 'A' && character <= 'Z') &&
			!(character >= '0' && character <= '9') &&
			!strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			return false
		}
	}
	return true
}

func splitHeaderList(value string) []string {
	var parts []string
	start := 0
	inQuotes := false
	escaped := false
	for index, character := range value {
		switch {
		case escaped:
			escaped = false
		case inQuotes && character == '\\':
			escaped = true
		case character == '"':
			inQuotes = !inQuotes
		case character == ',' && !inQuotes:
			parts = append(parts, value[start:index])
			start = index + 1
		}
	}
	return append(parts, value[start:])
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

func hasHeader(header map[string][]string, name string) bool {
	for existingName := range header {
		if strings.EqualFold(existingName, name) {
			return true
		}
	}
	return false
}

func headerValues(header map[string][]string, name string) []string {
	var values []string
	for existingName, existingValues := range header {
		if strings.EqualFold(existingName, name) {
			values = append(values, existingValues...)
		}
	}
	return values
}

func firstHeader(header map[string][]string, name string) string {
	values := headerValues(header, name)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func isCacheableContentType(contentType string) bool {
	cacheable := []string{"image/", "font/", "text/css", "text/javascript", "application/javascript", "application/x-javascript", "video/", "audio/"}
	for _, prefix := range cacheable {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}

func maxDuration(first, second time.Duration) time.Duration {
	if first > second {
		return first
	}
	return second
}
