package cdn

type CDN struct {
	ID       string
	Origin   string
	Domain   string
	IsActive bool
	CacheTTL uint
}
