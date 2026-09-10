package cache

import (
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
