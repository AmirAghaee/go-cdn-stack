package cache

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type requestCachePolicy struct {
	bypass       bool
	onlyIfCached bool
}

func evaluateRequestCachePolicy(request Request) requestCachePolicy {
	policy := requestCachePolicy{}
	if hasHeader(request.Header, "Authorization") ||
		hasHeader(request.Header, "Cookie") ||
		hasHeader(request.Header, "Range") ||
		hasConditionalHeader(request.Header) {
		policy.bypass = true
	}

	cacheControl, valid := parseCacheControl(request.Header)
	_, policy.onlyIfCached = cacheControl["only-if-cached"]
	if !policy.onlyIfCached {
		policy.onlyIfCached = hasCacheControlDirective(request.Header, "only-if-cached")
	}
	if !valid {
		policy.bypass = true
		return policy
	}
	for _, directive := range []string{"no-cache", "no-store", "max-age", "min-fresh"} {
		if _, found := cacheControl[directive]; found {
			policy.bypass = true
		}
	}

	for _, value := range headerValues(request.Header, "Pragma") {
		for _, directive := range splitHeaderList(value) {
			if strings.EqualFold(strings.TrimSpace(directive), "no-cache") {
				policy.bypass = true
			}
		}
	}

	return policy
}

func hasConditionalHeader(header map[string][]string) bool {
	for _, name := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "If-Range"} {
		if hasHeader(header, name) {
			return true
		}
	}
	return false
}

func hasCacheControlDirective(header map[string][]string, name string) bool {
	for _, value := range headerValues(header, "Cache-Control") {
		for _, part := range splitHeaderList(value) {
			directive, _, _ := strings.Cut(strings.TrimSpace(part), "=")
			if strings.EqualFold(strings.TrimSpace(directive), name) {
				return true
			}
		}
	}
	return false
}

type freshnessMetadata struct {
	expiresAt  time.Time
	storedAt   time.Time
	initialAge time.Duration
}

func cacheExpiry(now time.Time, configuredTTL uint, response Response) (time.Time, bool) {
	metadata, ok := cacheFreshness(now, now, configuredTTL, response)
	return metadata.expiresAt, ok
}

func cacheFreshness(requestTime, responseTime time.Time, configuredTTL uint, response Response) (freshnessMetadata, bool) {
	if response.StatusCode != http.StatusOK ||
		!isCacheableContentType(firstHeader(response.Header, "Content-Type")) ||
		hasHeader(response.Header, "Set-Cookie") ||
		!hasSupportedVary(response.Header) {
		return freshnessMetadata{}, false
	}

	configuredDuration, ok := cacheTTLDuration(configuredTTL)
	if !ok {
		return freshnessMetadata{}, false
	}
	configuredExpiry := responseTime.Add(configuredDuration)

	directives, valid := parseCacheControl(response.Header)
	if !valid {
		return freshnessMetadata{}, false
	}
	for _, directive := range []string{"private", "no-store", "no-cache"} {
		if _, found := directives[directive]; found {
			return freshnessMetadata{}, false
		}
	}

	initialAge, responseDate, ok := correctedInitialAge(requestTime, responseTime, response.Header)
	if !ok {
		return freshnessMetadata{}, false
	}
	metadata := freshnessMetadata{storedAt: responseTime, initialAge: initialAge}

	if values, found := directives["s-maxage"]; found {
		metadata.expiresAt, ok = boundedFreshnessExpiry(responseTime, configuredExpiry, values, initialAge)
		return metadata, ok
	}
	if values, found := directives["max-age"]; found {
		metadata.expiresAt, ok = boundedFreshnessExpiry(responseTime, configuredExpiry, values, initialAge)
		return metadata, ok
	}

	expiresValues := headerValues(response.Header, "Expires")
	if len(expiresValues) > 0 {
		if len(expiresValues) != 1 {
			return freshnessMetadata{}, false
		}
		expiresAt, err := http.ParseTime(strings.TrimSpace(expiresValues[0]))
		if err != nil {
			return freshnessMetadata{}, false
		}
		if !expiresAt.After(responseDate) {
			return freshnessMetadata{}, false
		}
		freshnessLifetime := expiresAt.Sub(responseDate)
		originExpiry := responseTime.Add(freshnessLifetime - initialAge)
		metadata.expiresAt, ok = earlierFutureExpiry(responseTime, configuredExpiry, originExpiry)
		return metadata, ok
	}

	metadata.expiresAt = configuredExpiry
	return metadata, true
}

func correctedInitialAge(requestTime, responseTime time.Time, header map[string][]string) (time.Duration, time.Time, bool) {
	ageValue, ok := parseAge(header)
	if !ok {
		return 0, time.Time{}, false
	}
	responseDelay := responseTime.Sub(requestTime)
	if responseDelay < 0 {
		responseDelay = 0
	}
	if ageValue > time.Duration(math.MaxInt64)-responseDelay {
		return 0, time.Time{}, false
	}
	correctedAgeValue := ageValue + responseDelay

	responseDate := responseTime
	dateValues := headerValues(header, "Date")
	if len(dateValues) > 0 {
		if len(dateValues) != 1 {
			return 0, time.Time{}, false
		}
		var err error
		responseDate, err = http.ParseTime(strings.TrimSpace(dateValues[0]))
		if err != nil {
			return 0, time.Time{}, false
		}
	}
	apparentAge := responseTime.Sub(responseDate)
	if apparentAge < 0 {
		apparentAge = 0
	}
	return maxDuration(apparentAge, correctedAgeValue), responseDate, true
}

func currentAge(now, storedAt time.Time, initialAge time.Duration) time.Duration {
	if storedAt.IsZero() || !now.After(storedAt) {
		return initialAge
	}
	residentTime := now.Sub(storedAt)
	if initialAge > time.Duration(math.MaxInt64)-residentTime {
		return time.Duration(math.MaxInt64)
	}
	return initialAge + residentTime
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
