package types

// cache_global group IDs registering, per company, the latest searchable change watermark that
// the ecommerce .db snapshot is rebuilt from. Exported because `business` builds the snapshot
// while `production` writes the products watermark on every catalog save.
const (
	CacheGroupProducts   = int16(1)
	CacheGroupBrands     = int16(2)
	CacheGroupCategories = int16(3)
)
