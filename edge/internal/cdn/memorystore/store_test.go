package memorystore

import (
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cdn"
)

func TestFindByDomainNormalizesHostAndRejectsInactiveCDN(t *testing.T) {
	active, err := cdn.New("active", "CDN.Example.", "http://origin.example", true, 60)
	if err != nil {
		t.Fatal(err)
	}
	inactive, err := cdn.New("inactive", "inactive.example", "http://origin.example", false, 60)
	if err != nil {
		t.Fatal(err)
	}
	store := New()
	store.Replace([]cdn.CDN{active, inactive})

	if _, ok := store.FindByDomain("cdn.example:443"); !ok {
		t.Fatal("active CDN was not found using a normalized request host")
	}
	if _, ok := store.FindByDomain("inactive.example"); ok {
		t.Fatal("inactive CDN was returned")
	}
}
