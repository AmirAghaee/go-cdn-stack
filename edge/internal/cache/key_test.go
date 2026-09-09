package cache

import (
	"testing"
)

func TestCacheKeyIncludesNormalizedAcceptEncoding(t *testing.T) {
	first := cacheKey("cdn.example", "/asset", map[string][]string{
		"Accept-Encoding": {"GZIP", " br "},
	})
	second := cacheKey("cdn.example", "/asset", map[string][]string{
		"accept-encoding": {"gzip,br"},
	})
	identity := cacheKey("cdn.example", "/asset", nil)

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
