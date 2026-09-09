package cache

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
