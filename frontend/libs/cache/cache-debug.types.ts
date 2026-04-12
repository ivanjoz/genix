export interface ICacheDebugRow {
	// Distinguishes row origin because delta and group caches represent different storage models.
	source: 'delta' | 'group'
	// Base route is the human-friendly grouping key shown in the first column.
	baseRoute: string
	// API route keeps the exact route/query-shape used by the cache entry.
	apiRoute: string
	// When exact persisted size is unavailable, the UI falls back to this record/group count.
	recordsCount: number
	// Size is optional because current cache schemas do not persist exact bytes per route.
	sizeMB?: number
}
