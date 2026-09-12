package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestEvaluateRequestCachePolicy(t *testing.T) {
	tests := map[string]struct {
		header           map[string][]string
		wantBypass       bool
		wantOnlyIfCached bool
	}{
		"ordinary request": {},
		"only if cached": {
			header:           map[string][]string{"Cache-Control": {"only-if-cached"}},
			wantOnlyIfCached: true,
		},
		"validator": {
			header:     map[string][]string{"If-None-Match": {`"asset-v1"`}},
			wantBypass: true,
		},
		"request max age": {
			header:     map[string][]string{"Cache-Control": {"max-age=10"}},
			wantBypass: true,
		},
		"request min fresh": {
			header:     map[string][]string{"Cache-Control": {"min-fresh=10"}},
			wantBypass: true,
		},
		"validation forbidden by only if cached": {
			header:           map[string][]string{"Cache-Control": {"only-if-cached, no-cache"}},
			wantBypass:       true,
			wantOnlyIfCached: true,
		},
		"only if cached survives malformed extension": {
			header:           map[string][]string{"Cache-Control": {`only-if-cached, extension="unterminated`}},
			wantBypass:       true,
			wantOnlyIfCached: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			policy := evaluateRequestCachePolicy(Request{Header: test.header})
			if policy.bypass != test.wantBypass || policy.onlyIfCached != test.wantOnlyIfCached {
				t.Fatalf("policy = %+v, want bypass=%t only-if-cached=%t", policy, test.wantBypass, test.wantOnlyIfCached)
			}
		})
	}
}

func TestCacheExpiryHonorsConfiguredAndOriginFreshness(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		ttl        uint
		header     map[string][]string
		wantOffset time.Duration
		wantOK     bool
	}{
		"configured ttl without origin freshness": {
			ttl:        120,
			header:     cacheableHeader(),
			wantOffset: 120 * time.Second,
			wantOK:     true,
		},
		"max age shortens configured ttl": {
			ttl:        120,
			header:     cacheableHeader("Cache-Control", "max-age=60"),
			wantOffset: 60 * time.Second,
			wantOK:     true,
		},
		"configured ttl caps max age": {
			ttl:        30,
			header:     cacheableHeader("Cache-Control", "max-age=60"),
			wantOffset: 30 * time.Second,
			wantOK:     true,
		},
		"s maxage takes precedence": {
			ttl:        120,
			header:     cacheableHeader("Cache-Control", "max-age=90, s-maxage=45"),
			wantOffset: 45 * time.Second,
			wantOK:     true,
		},
		"quoted max age": {
			ttl:        120,
			header:     cacheableHeader("Cache-Control", `public, max-age="40"`),
			wantOffset: 40 * time.Second,
			wantOK:     true,
		},
		"age reduces remaining freshness": {
			ttl:        120,
			header:     cacheableHeader("Cache-Control", "max-age=60", "Age", "15"),
			wantOffset: 45 * time.Second,
			wantOK:     true,
		},
		"date apparent age reduces max age": {
			ttl: 120,
			header: cacheableHeader(
				"Cache-Control", "max-age=60",
				"Date", now.Add(-20*time.Second).Format(http.TimeFormat),
			),
			wantOffset: 40 * time.Second,
			wantOK:     true,
		},
		"expires and date define freshness": {
			ttl: 120,
			header: cacheableHeader(
				"Date", now.Add(-10*time.Second).Format(http.TimeFormat),
				"Expires", now.Add(50*time.Second).Format(http.TimeFormat),
			),
			wantOffset: 50 * time.Second,
			wantOK:     true,
		},
		"age can exceed apparent age": {
			ttl: 120,
			header: cacheableHeader(
				"Date", now.Add(-10*time.Second).Format(http.TimeFormat),
				"Expires", now.Add(50*time.Second).Format(http.TimeFormat),
				"Age", "20",
			),
			wantOffset: 40 * time.Second,
			wantOK:     true,
		},
		"zero ttl": {
			ttl:    0,
			header: cacheableHeader(),
		},
		"zero max age": {
			ttl:    120,
			header: cacheableHeader("Cache-Control", "max-age=0"),
		},
		"expired response": {
			ttl:    120,
			header: cacheableHeader("Expires", now.Add(-time.Second).Format(http.TimeFormat)),
		},
		"malformed max age": {
			ttl:    120,
			header: cacheableHeader("Cache-Control", "max-age=invalid"),
		},
		"malformed cache control quoted string": {
			ttl:    120,
			header: cacheableHeader("Cache-Control", `extension="unterminated, no-store`),
		},
		"conflicting max age": {
			ttl:    120,
			header: cacheableHeader("Cache-Control", "max-age=60, max-age=30"),
		},
		"malformed expires": {
			ttl:    120,
			header: cacheableHeader("Expires", "tomorrow"),
		},
		"expires before date": {
			ttl: 120,
			header: cacheableHeader(
				"Date", now.Format(http.TimeFormat),
				"Expires", now.Add(-time.Second).Format(http.TimeFormat),
			),
		},
		"malformed age": {
			ttl:    120,
			header: cacheableHeader("Age", "old"),
		},
		"malformed date": {
			ttl:    120,
			header: cacheableHeader("Date", "yesterday"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expiresAt, ok := cacheExpiry(now, test.ttl, Response{
				StatusCode: http.StatusOK,
				Header:     test.header,
			})
			if ok != test.wantOK {
				t.Fatalf("cacheExpiry() ok = %t, want %t", ok, test.wantOK)
			}
			if ok && !expiresAt.Equal(now.Add(test.wantOffset)) {
				t.Fatalf("expiry = %s, want %s", expiresAt, now.Add(test.wantOffset))
			}
		})
	}
}

func TestCacheFreshnessIncludesOriginResponseDelay(t *testing.T) {
	responseTime := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	requestTime := responseTime.Add(-5 * time.Second)
	response := Response{
		StatusCode: http.StatusOK,
		Header: cacheableHeader(
			"Cache-Control", "max-age=60",
			"Date", responseTime.Format(http.TimeFormat),
			"Age", "10",
		),
	}

	metadata, ok := cacheFreshness(requestTime, responseTime, 120, response)
	if !ok {
		t.Fatal("cacheFreshness() rejected cacheable response")
	}
	if metadata.initialAge != 15*time.Second || metadata.storedAt != responseTime ||
		!metadata.expiresAt.Equal(responseTime.Add(45*time.Second)) {
		t.Fatalf("freshness metadata = %+v", metadata)
	}
}

func TestCurrentAgeAddsResidentTime(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 30, 0, time.UTC)
	if got := currentAge(now, now.Add(-20*time.Second), 15*time.Second); got != 35*time.Second {
		t.Fatalf("currentAge() = %v, want 35s", got)
	}
}

func TestCacheExpiryAcceptsOnlySupportedVaryDimensions(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	tests := map[string]struct {
		vary   []string
		wantOK bool
	}{
		"no vary":                  {wantOK: true},
		"accept encoding":          {vary: []string{"Accept-Encoding"}, wantOK: true},
		"repeated accept encoding": {vary: []string{"accept-encoding", "ACCEPT-ENCODING"}, wantOK: true},
		"unsupported":              {vary: []string{"Accept-Language"}},
		"mixed":                    {vary: []string{"Accept-Encoding, Accept-Language"}},
		"wildcard":                 {vary: []string{"*"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			header := cacheableHeader()
			if test.vary != nil {
				header["vArY"] = test.vary
			}
			_, ok := cacheExpiry(now, 60, Response{StatusCode: http.StatusOK, Header: header})
			if ok != test.wantOK {
				t.Fatalf("cacheExpiry() ok = %t, want %t", ok, test.wantOK)
			}
		})
	}
}

func cacheableHeader(values ...string) map[string][]string {
	header := map[string][]string{"Content-Type": {"image/png"}}
	for index := 0; index < len(values); index += 2 {
		header[values[index]] = append(header[values[index]], values[index+1])
	}
	return header
}
