package cache

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

func TestCacheKeyIncludesNormalizedAcceptEncoding(t *testing.T) {
	item := mustCDN(t, 60)
	first := cacheKey(item, "/asset", map[string][]string{
		"Accept-Encoding": {"GZIP", " br "},
	})
	second := cacheKey(item, "/asset", map[string][]string{
		"accept-encoding": {"gzip,br"},
	})
	identity := cacheKey(item, "/asset", nil)

	if first != second {
		t.Fatalf("equivalent Accept-Encoding values produced different keys:\n%s\n%s", first, second)
	}
	if first == identity {
		t.Fatal("encoded and identity representations produced the same key")
	}
	if first[:len(cacheKeyVersion)] != cacheKeyVersion {
		t.Fatalf("cache key %q does not start with version %q", first, cacheKeyVersion)
	}
}

func TestDomainFromKey(t *testing.T) {
	item, err := cdn.New("id-1", "CDN.Example.", "http://origin.example", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	key := cacheKey(item, "/asset|host=999:wrong?x=1", map[string][]string{
		"Accept-Encoding": {"gzip, br"},
	})

	domain, ok := DomainFromKey(key)
	if !ok || domain != "cdn.example" {
		t.Fatalf("DomainFromKey() = %q, %v; want cdn.example, true", domain, ok)
	}
}

func TestDomainFromKeyRejectsMalformedKeys(t *testing.T) {
	configuration := strings.Repeat("0", 64)
	cases := []string{
		"",
		"v2|configuration=" + configuration + "|host=11:cdn.example|uri=1:/|accept-encoding=0:",
		"v3|configuration=invalid|host=11:cdn.example|uri=1:/|accept-encoding=0:",
		"v3|configuration=" + configuration + "|host=99:cdn.example|uri=1:/|accept-encoding=0:",
		"v3|configuration=" + configuration + "|host=11:CDN.EXAMPLE|uri=1:/|accept-encoding=0:",
		"v3|configuration=" + configuration + "|host=11:cdn.example|uri=2:/|accept-encoding=0:",
		"v3|configuration=" + configuration + "|host=11:cdn.example|uri=1:/|accept-encoding=0:trailing",
	}
	for index, key := range cases {
		t.Run(fmt.Sprintf("case-%d", index), func(t *testing.T) {
			if domain, ok := DomainFromKey(key); ok {
				t.Fatalf("DomainFromKey(%q) = %q, true; want false", key, domain)
			}
		})
	}
}

func TestCacheKeyChangesWithCDNConfiguration(t *testing.T) {
	base, err := cdn.New("id-1", "cdn.example", "http://origin.example", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]cdn.CDN{}
	for name, values := range map[string]struct {
		id     string
		origin string
		ttl    uint
	}{
		"identity": {id: "id-2", origin: "http://origin.example", ttl: 60},
		"origin":   {id: "id-1", origin: "http://other-origin.example", ttl: 60},
		"TTL":      {id: "id-1", origin: "http://origin.example", ttl: 0},
	} {
		item, createErr := cdn.New(values.id, "cdn.example", values.origin, true, values.ttl)
		if createErr != nil {
			t.Fatal(createErr)
		}
		cases[name] = item
	}

	baseKey := cacheKey(base, "/asset", nil)
	for name, item := range cases {
		t.Run(name, func(t *testing.T) {
			if got := cacheKey(item, "/asset", nil); got == baseKey {
				t.Fatalf("configuration change produced unchanged cache key %q", got)
			}
		})
	}
}
