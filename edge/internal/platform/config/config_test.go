package config

import (
	"reflect"
	"testing"
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
