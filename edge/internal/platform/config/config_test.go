package config

import (
	"reflect"
	"testing"
	"time"
)

func TestTrustedProxiesEnvironment(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  []string
	}{
		{"", []string{}},
		{"10.0.0.0/8,2001:db8::/32", []string{"10.0.0.0/8", "2001:db8::/32"}},
	} {
		t.Setenv("TRUSTED_PROXIES", tt.value)
		if got := Load().TrustedProxies; !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("TRUSTED_PROXIES=%q: got %v want %v", tt.value, got, tt.want)
		}
	}
}

func TestHTTPAndOriginLimitsEnvironment(t *testing.T) {
	values := map[string]string{
		"HTTP_READ_HEADER_TIMEOUT":             "7",
		"HTTP_IDLE_TIMEOUT":                    "80",
		"HTTP_MAX_HEADER_BYTES":                "2048",
		"HTTP_MAX_CONNECTIONS":                 "18",
		"HTTP_MAX_CONCURRENT_REQUESTS":         "12",
		"ORIGIN_DIAL_TIMEOUT":                  "3",
		"ORIGIN_RESPONSE_HEADER_TIMEOUT":       "4",
		"ORIGIN_IDLE_CONNECTION_TIMEOUT":       "50",
		"ORIGIN_MAX_IDLE_CONNECTIONS":          "20",
		"ORIGIN_MAX_IDLE_CONNECTIONS_PER_HOST": "5",
		"ORIGIN_MAX_CONCURRENT_REQUESTS":       "9",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}

	cfg := Load()
	if cfg.HTTPReadHeaderTimeoutDuration != 7*time.Second ||
		cfg.HTTPIdleTimeoutDuration != 80*time.Second ||
		cfg.HTTPMaxHeaderBytes != 2048 || cfg.HTTPMaxConnections != 18 || cfg.HTTPMaxConcurrent != 12 {
		t.Fatalf("HTTP limits = %+v", cfg)
	}
	if cfg.OriginDialTimeoutDuration != 3*time.Second ||
		cfg.OriginResponseHeaderTimeoutDuration != 4*time.Second ||
		cfg.OriginIdleConnTimeoutDuration != 50*time.Second ||
		cfg.OriginMaxIdleConns != 20 || cfg.OriginMaxIdlePerHost != 5 || cfg.OriginMaxConcurrent != 9 {
		t.Fatalf("origin limits = %+v", cfg)
	}
}

func TestOriginAllowedCIDRsEnvironment(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  []string
	}{
		{"", []string{}},
		{"10.20.30.0/24,fd00:1234::/48", []string{"10.20.30.0/24", "fd00:1234::/48"}},
	} {
		t.Setenv("ORIGIN_ALLOWED_CIDRS", tt.value)
		if got := Load().OriginAllowedCIDRs; !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("ORIGIN_ALLOWED_CIDRS=%q: got %v want %v", tt.value, got, tt.want)
		}
	}
}
