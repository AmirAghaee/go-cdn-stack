package repository

import (
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
)

func TestGetByDomainOnlyReturnsActiveCDNs(t *testing.T) {
	repo := NewCdnRepository()
	repo.Set([]domain.CDN{
		{Domain: "active.example", IsActive: true},
		{Domain: "inactive.example", IsActive: false},
	})

	if _, ok := repo.GetByDomain("active.example"); !ok {
		t.Fatal("active CDN was not returned")
	}
	if _, ok := repo.GetByDomain("inactive.example"); ok {
		t.Fatal("inactive CDN was returned")
	}
}
