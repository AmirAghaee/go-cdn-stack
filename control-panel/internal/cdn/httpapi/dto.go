package httpapi

import "github.com/AmirAghaee/go-cdn-stack/control-panel/internal/cdn"

type cdnRequest struct {
	Origin   string `json:"origin" binding:"required,url"`
	Domain   string `json:"domain" binding:"required"`
	IsActive bool   `json:"is_active"`
	CacheTTL uint   `json:"cache_ttl"`
}

type cdnResponse struct {
	ID       string `json:"id"`
	Origin   string `json:"origin"`
	Domain   string `json:"domain"`
	IsActive bool   `json:"is_active"`
	CacheTTL uint   `json:"cache_ttl"`
}

func newCDNResponse(item *cdn.CDN) cdnResponse {
	return cdnResponse{
		ID:       item.ID,
		Origin:   item.Origin,
		Domain:   item.Domain,
		IsActive: item.IsActive,
		CacheTTL: item.CacheTTL,
	}
}
