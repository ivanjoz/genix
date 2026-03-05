package db

import (
	"fmt"
	"runtime"
	"testing"
	"unsafe"
)

// getMemStats returns current memory stats after GC
func getMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m
}

// formatBytes formats bytes into human-readable string
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Mock table types to avoid import cycles
// SimpleTable - minimal table with 5 columns
type simpleRecord struct {
	TableStruct[simpleSchema, simpleRecord]
	EmpresaID int32  `db:"empresa_id"`
	ID        string `db:"id"`
	Name      string `db:"name"`
	Updated   int64  `db:"updated"`
	Status    int8   `db:"status"`
}

type simpleSchema struct {
	TableStruct[simpleSchema, simpleRecord]
	EmpresaID Col[simpleSchema, int32]
	ID        Col[simpleSchema, string]
	Name      Col[simpleSchema, string]
	Updated   Col[simpleSchema, int64]
	Status    Col[simpleSchema, int8]
}

func (e simpleSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "simple_table",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

// MediumTable - medium complexity with 15 columns and 2 indexes
type mediumRecord struct {
	TableStruct[mediumSchema, mediumRecord]
	EmpresaID      int32   `db:"empresa_id"`
	ID             string  `db:"id"`
	SKU            string  `db:"sku"`
	Name           string  `db:"name"`
	Description    string  `db:"description"`
	Price          float64 `db:"price"`
	Quantity       int32   `db:"quantity"`
	CategoryID     int32   `db:"category_id"`
	SubcategoryID  int32   `db:"subcategory_id"`
	BrandID        int32   `db:"brand_id"`
	SupplierID     int32   `db:"supplier_id"`
	Weight         float64 `db:"weight"`
	Active         bool    `db:"active"`
	Updated        int64   `db:"updated"`
	Status         int8    `db:"status"`
}

type mediumSchema struct {
	TableStruct[mediumSchema, mediumRecord]
	EmpresaID     Col[mediumSchema, int32]
	ID            Col[mediumSchema, string]
	SKU           Col[mediumSchema, string]
	Name          Col[mediumSchema, string]
	Description   Col[mediumSchema, string]
	Price         Col[mediumSchema, float64]
	Quantity      Col[mediumSchema, int32]
	CategoryID    Col[mediumSchema, int32]
	SubcategoryID Col[mediumSchema, int32]
	BrandID       Col[mediumSchema, int32]
	SupplierID    Col[mediumSchema, int32]
	Weight        Col[mediumSchema, float64]
	Active        Col[mediumSchema, bool]
	Updated       Col[mediumSchema, int64]
	Status        Col[mediumSchema, int8]
}

func (e mediumSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:         "medium_table",
		Partition:    e.EmpresaID,
		Keys:         []Coln{e.ID},
		LocalIndexes: []Coln{e.SKU, e.CategoryID},
		Views: []View{
			{
				Cols:     []Coln{e.CategoryID, e.Status},
				KeepPart: true,
			},
		},
	}
}

// ComplexTable - complex table with 25 columns, multiple indexes, and views
type complexRecord struct {
	TableStruct[complexSchema, complexRecord]
	EmpresaID       int32    `db:"empresa_id"`
	ID              string   `db:"id"`
	Code            string   `db:"code"`
	Reference       string   `db:"reference"`
	Name            string   `db:"name"`
	Description     string   `db:"description"`
	CategoryID      int32    `db:"category_id"`
	SubcategoryID   int32    `db:"subcategory_id"`
	BrandID         int32    `db:"brand_id"`
	SupplierID      int32    `db:"supplier_id"`
	WarehouseID     int32    `db:"warehouse_id"`
	LocationID      int32    `db:"location_id"`
	Price           float64  `db:"price"`
	Cost            float64  `db:"cost"`
	Quantity        int32    `db:"quantity"`
	MinQuantity     int32    `db:"min_quantity"`
	MaxQuantity     int32    `db:"max_quantity"`
	Weight          float64  `db:"weight"`
	Volume          float64  `db:"volume"`
	TaxRate         float64  `db:"tax_rate"`
	DiscountPercent float64  `db:"discount_percent"`
	Tags            []string `db:"tags"`
	Attributes      []string `db:"attributes"`
	Updated         int64    `db:"updated"`
	Status          int8     `db:"status"`
	Active          bool     `db:"active"`
}

type complexSchema struct {
	TableStruct[complexSchema, complexRecord]
	EmpresaID       Col[complexSchema, int32]
	ID              Col[complexSchema, string]
	Code            Col[complexSchema, string]
	Reference       Col[complexSchema, string]
	Name            Col[complexSchema, string]
	Description     Col[complexSchema, string]
	CategoryID      Col[complexSchema, int32]
	SubcategoryID   Col[complexSchema, int32]
	BrandID         Col[complexSchema, int32]
	SupplierID      Col[complexSchema, int32]
	WarehouseID     Col[complexSchema, int32]
	LocationID      Col[complexSchema, int32]
	Price           Col[complexSchema, float64]
	Cost            Col[complexSchema, float64]
	Quantity        Col[complexSchema, int32]
	MinQuantity     Col[complexSchema, int32]
	MaxQuantity     Col[complexSchema, int32]
	Weight          Col[complexSchema, float64]
	Volume          Col[complexSchema, float64]
	TaxRate         Col[complexSchema, float64]
	DiscountPercent Col[complexSchema, float64]
	Tags            Col[complexSchema, []string]
	Attributes      Col[complexSchema, []string]
	Updated         Col[complexSchema, int64]
	Status          Col[complexSchema, int8]
	Active          Col[complexSchema, bool]
}

func (e complexSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:         "complex_table",
		Partition:    e.EmpresaID,
		Keys:         []Coln{e.ID},
		LocalIndexes: []Coln{e.Code, e.Reference, e.CategoryID, e.WarehouseID},
		Indexes:      [][]Coln{{e.SupplierID}, {e.BrandID}},
		Views: []View{
			{
				Cols:     []Coln{e.CategoryID, e.Status},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.WarehouseID, e.Updated},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.SupplierID, e.Active},
				KeepPart: true,
			},
		},
	}
}

// VeryComplexTable - very complex table with 40 columns, many indexes, views, and KeyConcatenated
type veryComplexRecord struct {
	TableStruct[veryComplexSchema, veryComplexRecord]
	EmpresaID         int32    `db:"empresa_id"`
	ID                string   `db:"id"`
	Code              string   `db:"code"`
	Reference         string   `db:"reference"`
	Name              string   `db:"name"`
	Description       string   `db:"description"`
	LongDescription   string   `db:"long_description"`
	Notes             string   `db:"notes"`
	CategoryID        int32    `db:"category_id"`
	SubcategoryID     int32    `db:"subcategory_id"`
	BrandID           int32    `db:"brand_id"`
	SupplierID        int32    `db:"supplier_id"`
	WarehouseID       int32    `db:"warehouse_id"`
	LocationID        int32    `db:"location_id"`
	ZoneID            int32    `db:"zone_id"`
	RegionID          int32    `db:"region_id"`
	Price             float64  `db:"price"`
	Cost              float64  `db:"cost"`
	Quantity          int32    `db:"quantity"`
	MinQuantity       int32    `db:"min_quantity"`
	MaxQuantity       int32    `db:"max_quantity"`
	ReservedQuantity  int32    `db:"reserved_quantity"`
	OrderedQuantity   int32    `db:"ordered_quantity"`
	Weight            float64  `db:"weight"`
	Volume            float64  `db:"volume"`
	TaxRate           float64  `db:"tax_rate"`
	DiscountPercent   float64  `db:"discount_percent"`
	Margin            float64  `db:"margin"`
	CommissionPercent float64  `db:"commission_percent"`
	Tags              []string `db:"tags"`
	Attributes        []string `db:"attributes"`
	CustomFields      []string `db:"custom_fields"`
	ImageURLs         []string `db:"image_urls"`
	RelatedProducts   []string `db:"related_products"`
	Created           int64    `db:"created"`
	Updated           int64    `db:"updated"`
	LastPurchaseDate  int64    `db:"last_purchase_date"`
	LastSaleDate      int64    `db:"last_sale_date"`
	Status            int8     `db:"status"`
	Active            bool     `db:"active"`
}

type veryComplexSchema struct {
	TableStruct[veryComplexSchema, veryComplexRecord]
	EmpresaID         Col[veryComplexSchema, int32]
	ID                Col[veryComplexSchema, string]
	Code              Col[veryComplexSchema, string]
	Reference         Col[veryComplexSchema, string]
	Name              Col[veryComplexSchema, string]
	Description       Col[veryComplexSchema, string]
	LongDescription   Col[veryComplexSchema, string]
	Notes             Col[veryComplexSchema, string]
	CategoryID        Col[veryComplexSchema, int32]
	SubcategoryID     Col[veryComplexSchema, int32]
	BrandID           Col[veryComplexSchema, int32]
	SupplierID        Col[veryComplexSchema, int32]
	WarehouseID       Col[veryComplexSchema, int32]
	LocationID        Col[veryComplexSchema, int32]
	ZoneID            Col[veryComplexSchema, int32]
	RegionID          Col[veryComplexSchema, int32]
	Price             Col[veryComplexSchema, float64]
	Cost              Col[veryComplexSchema, float64]
	Quantity          Col[veryComplexSchema, int32]
	MinQuantity       Col[veryComplexSchema, int32]
	MaxQuantity       Col[veryComplexSchema, int32]
	ReservedQuantity  Col[veryComplexSchema, int32]
	OrderedQuantity   Col[veryComplexSchema, int32]
	Weight            Col[veryComplexSchema, float64]
	Volume            Col[veryComplexSchema, float64]
	TaxRate           Col[veryComplexSchema, float64]
	DiscountPercent   Col[veryComplexSchema, float64]
	Margin            Col[veryComplexSchema, float64]
	CommissionPercent Col[veryComplexSchema, float64]
	Tags              Col[veryComplexSchema, []string]
	Attributes        Col[veryComplexSchema, []string]
	CustomFields      Col[veryComplexSchema, []string]
	ImageURLs         Col[veryComplexSchema, []string]
	RelatedProducts   Col[veryComplexSchema, []string]
	Created           Col[veryComplexSchema, int64]
	Updated           Col[veryComplexSchema, int64]
	LastPurchaseDate  Col[veryComplexSchema, int64]
	LastSaleDate      Col[veryComplexSchema, int64]
	Status            Col[veryComplexSchema, int8]
	Active            Col[veryComplexSchema, bool]
}

func (e veryComplexSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:            "very_complex_table",
		Partition:       e.EmpresaID,
		Keys:            []Coln{e.ID},
		KeyConcatenated: []Coln{e.WarehouseID, e.CategoryID, e.SupplierID},
		LocalIndexes:    []Coln{e.Code, e.Reference, e.CategoryID, e.WarehouseID, e.ZoneID, e.RegionID},
		Indexes:         [][]Coln{{e.SupplierID}, {e.BrandID}, {e.LocationID}},
		Views: []View{
			{
				Cols:     []Coln{e.CategoryID, e.Status},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.WarehouseID, e.Updated},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.SupplierID, e.Active},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.ZoneID, e.Quantity},
				KeepPart: true,
			},
			{
				Cols:     []Coln{e.RegionID, e.Price},
				KeepPart: true,
			},
		},
	}
}

// TestMemoryUsagePerTable measures memory usage when compiling multiple tables
func TestMemoryUsagePerTable(t *testing.T) {
	// Reset caches to start fresh
	resetORMTableCachesForTesting()

	// Force GC before starting
	runtime.GC()

	// Measure baseline memory
	before := getMemStats()
	baselineAlloc := before.Alloc

	t.Logf("=== Baseline Memory Usage ===")
	t.Logf("Alloc: %s", formatBytes(before.Alloc))
	t.Logf("TotalAlloc: %s", formatBytes(before.TotalAlloc))
	t.Logf("Sys: %s", formatBytes(before.Sys))
	t.Logf("NumGC: %d", before.NumGC)

	// Define mock table types with varying complexity
	tableTests := []struct {
		name string
		fn   func() ScyllaTable[any]
	}{
		{
			name: "SimpleTable (5 columns)",
			fn:   func() ScyllaTable[any] { return MakeScyllaTable[simpleRecord, simpleSchema]() },
		},
		{
			name: "MediumTable (15 columns, 2 indexes)",
			fn:   func() ScyllaTable[any] { return MakeScyllaTable[mediumRecord, mediumSchema]() },
		},
		{
			name: "ComplexTable (26 columns, 4 local indexes, 2 global indexes, 3 views)",
			fn:   func() ScyllaTable[any] { return MakeScyllaTable[complexRecord, complexSchema]() },
		},
		{
			name: "VeryComplexTable (40 columns, 6 local indexes, 3 global indexes, 5 views, KeyConcatenated)",
			fn:   func() ScyllaTable[any] { return MakeScyllaTable[veryComplexRecord, veryComplexSchema]() },
		},
	}

	// Compile tables one by one and measure memory after each
	for i, tt := range tableTests {
		// Force GC before measurement
		runtime.GC()

		// Measure memory before this table
		beforeTable := getMemStats()

		// Compile the table
		table := tt.fn()

		// Force GC after compilation
		runtime.GC()

		// Measure memory after this table
		afterTable := getMemStats()

		// Calculate delta
		delta := int64(afterTable.Alloc) - int64(beforeTable.Alloc)

		t.Logf("\n=== Table %d: %s ===", i+1, tt.name)
		t.Logf("Columns: %d", len(table.GetColumns()))
		t.Logf("Memory delta: %s", formatBytes(uint64(delta)))
		t.Logf("Current Alloc: %s", formatBytes(afterTable.Alloc))
		t.Logf("Cumulative Alloc: %s", formatBytes(afterTable.Alloc-baselineAlloc))

		// Additional table statistics
		if delta > 0 {
			bytesPerColumn := delta / int64(len(table.GetColumns()))
			t.Logf("Average per column: %s", formatBytes(uint64(bytesPerColumn)))
		}

		// Warn if memory per table seems high
		if delta > 1*1024*1024 { // 1 MB
			t.Logf("WARNING: Table consumed %s of memory!", formatBytes(uint64(delta)))
		}
	}

	// Final memory stats
	runtime.GC()
	after := getMemStats()
	totalDelta := int64(after.Alloc) - int64(baselineAlloc)

	t.Logf("\n=== Final Memory Summary ===")
	t.Logf("Total memory used for %d tables: %s", len(tableTests), formatBytes(uint64(totalDelta)))
	t.Logf("Average per table: %s", formatBytes(uint64(totalDelta/int64(len(tableTests)))))
	t.Logf("Current Alloc: %s", formatBytes(after.Alloc))
	t.Logf("TotalAlloc: %s", formatBytes(after.TotalAlloc))
	t.Logf("Sys: %s", formatBytes(after.Sys))
	t.Logf("NumGC: %d", after.NumGC)
}

// TestStructSizes measures the size of various ORM structures
func TestStructSizes(t *testing.T) {
	t.Log("=== Structure Sizes ===")

	// Measure struct sizes using unsafe.Sizeof
	t.Logf("colInfo size: %d bytes", unsafe.Sizeof(colInfo{}))
	t.Logf("columnInfo size: %d bytes", unsafe.Sizeof(columnInfo{}))
	t.Logf("viewInfo size: %d bytes", unsafe.Sizeof(viewInfo{}))
	t.Logf("QueryCapability size: %d bytes", unsafe.Sizeof(QueryCapability{}))

	// Create a sample columnInfo to measure actual allocation
	sampleCol := &columnInfo{
		colInfo: colInfo{
			Name:      "test_column",
			FieldName: "TestColumn",
			Idx:       1,
		},
		colType: colType{
			Type:      1,
			FieldType: "string",
			ColType:   "text",
		},
	}

	t.Logf("\nBefore compiling accessors:")
	t.Logf("columnInfo struct: %d bytes", unsafe.Sizeof(*sampleCol))

	// Compile accessors to see the impact
	sampleCol.compileFastAccessors()

	t.Logf("\nAfter compiling accessors:")
	t.Logf("columnInfo struct: %d bytes", unsafe.Sizeof(*sampleCol))
	t.Logf("Function pointer size: %d bytes", unsafe.Sizeof(sampleCol.getValue))

	// Test interface overhead
	var iColInfo IColInfo = sampleCol
	t.Logf("IColInfo interface size: %d bytes", unsafe.Sizeof(iColInfo))

	// Measure actual memory allocation for a columnInfo with closures
	runtime.GC()
	before := getMemStats()

	const numColumns = 100
	columns := make([]*columnInfo, numColumns)
	for i := 0; i < numColumns; i++ {
		col := &columnInfo{
			colInfo: colInfo{
				Name:      fmt.Sprintf("column_%d", i),
				FieldName: fmt.Sprintf("Column%d", i),
				Idx:       int16(i),
			},
			colType: colType{
				Type:      1,
				FieldType: "string",
				ColType:   "text",
			},
		}
		col.compileFastAccessors()
		columns[i] = col
	}

	runtime.GC()
	after := getMemStats()

	delta := int64(after.Alloc) - int64(before.Alloc)
	perColumn := delta / numColumns

	t.Logf("\n=== Memory for %d columns with closures ===", numColumns)
	t.Logf("Total memory: %s", formatBytes(uint64(delta)))
	t.Logf("Per column: %s (%d bytes)", formatBytes(uint64(perColumn)), perColumn)
}

// TestClosureMemoryOverhead measures memory overhead of closures
func TestClosureMemoryOverhead(t *testing.T) {
	t.Log("=== Closure Memory Overhead ===")

	runtime.GC()
	before := getMemStats()

	// Create many closures to measure overhead
	const numClosures = 10000
	closures := make([]func() string, numClosures)

	for i := 0; i < numClosures; i++ {
		// Simulate closure that captures a variable (like in compileFastAccessors)
		captured := fmt.Sprintf("value_%d", i)
		closures[i] = func() string {
			return captured
		}
	}

	runtime.GC()
	after := getMemStats()

	delta := int64(after.Alloc) - int64(before.Alloc)
	perClosure := delta / numClosures

	t.Logf("Created %d closures", numClosures)
	t.Logf("Total memory: %s", formatBytes(uint64(delta)))
	t.Logf("Per closure: %s (%d bytes)", formatBytes(uint64(perClosure)), perClosure)

	// Estimate for typical table
	columnsPerTable := 30
	closuresPerColumn := 4
	totalClosuresPerTable := columnsPerTable * closuresPerColumn
	estimatedMemory := perClosure * int64(totalClosuresPerTable)

	t.Logf("\nEstimated memory for 1 table (%d columns, %d closures/column):",
		columnsPerTable, closuresPerColumn)
	t.Logf("Closures per table: %d", totalClosuresPerTable)
	t.Logf("Estimated memory: %s", formatBytes(uint64(estimatedMemory)))
}

// TestMapMemoryOverhead measures memory overhead of maps in ScyllaTable
func TestMapMemoryOverhead(t *testing.T) {
	t.Log("=== Map Memory Overhead ===")

	runtime.GC()
	before := getMemStats()

	// Create maps similar to those in ScyllaTable
	const numEntries = 30

	stringToInterface := make(map[string]IColInfo, numEntries)
	intToInterface := make(map[int16]IColInfo, numEntries)
	stringToViewInfo := make(map[string]*viewInfo, numEntries)

	// Populate maps
	for i := 0; i < numEntries; i++ {
		col := &columnInfo{
			colInfo: colInfo{
				Name: fmt.Sprintf("column_%d", i),
				Idx:  int16(i),
			},
		}
		stringToInterface[col.Name] = col
		intToInterface[int16(i)] = col
	}

	for i := 0; i < 5; i++ {
		view := &viewInfo{
			name: fmt.Sprintf("index_%d", i),
			Type: 1,
		}
		stringToViewInfo[view.name] = view
	}

	runtime.GC()
	after := getMemStats()

	delta := int64(after.Alloc) - int64(before.Alloc)

	t.Logf("Created maps with %d column entries and 5 view entries", numEntries)
	t.Logf("Total map memory: %s", formatBytes(uint64(delta)))
	t.Logf("Per entry: %s", formatBytes(uint64(delta/numEntries)))
}

// TestDetailedTableMemoryBreakdown provides detailed breakdown of memory usage per table
func TestDetailedTableMemoryBreakdown(t *testing.T) {
	resetORMTableCachesForTesting()
	runtime.GC()

	t.Log("=== Detailed Table Memory Breakdown ===")

	// Test each table type in detail
	tables := []struct {
		name string
		fn   func() ScyllaTable[any]
	}{
		{"SimpleTable", func() ScyllaTable[any] { return MakeScyllaTable[simpleRecord, simpleSchema]() }},
		{"MediumTable", func() ScyllaTable[any] { return MakeScyllaTable[mediumRecord, mediumSchema]() }},
		{"ComplexTable", func() ScyllaTable[any] { return MakeScyllaTable[complexRecord, complexSchema]() }},
		{"VeryComplexTable", func() ScyllaTable[any] { return MakeScyllaTable[veryComplexRecord, veryComplexSchema]() }},
	}

	for _, tt := range tables {
		resetORMTableCachesForTesting()
		runtime.GC()

		before := getMemStats()
		table := tt.fn()
		runtime.GC()
		after := getMemStats()

		totalDelta := int64(after.Alloc) - int64(before.Alloc)

		t.Logf("\n--- %s ---", tt.name)
		t.Logf("Total memory: %s (%d bytes)", formatBytes(uint64(totalDelta)), totalDelta)
		t.Logf("Columns: %d", len(table.GetColumns()))
		t.Logf("Keys: %d", len(table.GetKeys()))

		// Estimate breakdown
		numColumns := len(table.GetColumns())
		if numColumns > 0 {
			perColumn := totalDelta / int64(numColumns)
			t.Logf("Average per column: %s (%d bytes)", formatBytes(uint64(perColumn)), perColumn)

			// Estimate closures (4 per column)
			estimatedClosures := int64(numColumns * 4 * 80) // ~80 bytes per closure
			t.Logf("Estimated closures (4 per column × 80 bytes): %s", formatBytes(uint64(estimatedClosures)))

			// Estimate structs
			estimatedStructs := int64(numColumns) * int64(unsafe.Sizeof(columnInfo{}))
			t.Logf("Estimated columnInfo structs: %s", formatBytes(uint64(estimatedStructs)))

			// Other overhead
			otherOverhead := totalDelta - estimatedClosures - estimatedStructs
			t.Logf("Other overhead (maps, indexes, views): %s", formatBytes(uint64(otherOverhead)))
		}
	}
}

// BenchmarkTableCompilationMemory benchmarks memory allocation during table compilation
func BenchmarkTableCompilationMemory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resetORMTableCachesForTesting()
		runtime.GC()
		b.StartTimer()

		// Compile a table
		_ = MakeScyllaTable[complexRecord, complexSchema]()
	}
}

// BenchmarkTableCompilationMemoryWarm benchmarks memory with warm cache (should be 0 allocations)
func BenchmarkTableCompilationMemoryWarm(b *testing.B) {
	// Pre-compile the table
	_ = MakeScyllaTable[complexRecord, complexSchema]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = MakeScyllaTable[complexRecord, complexSchema]()
	}
}

// TestMultipleTableMemoryGrowth tests memory growth when compiling many different tables
func TestMultipleTableMemoryGrowth(t *testing.T) {
	// This test simulates your scenario: compiling multiple different tables
	// to see if memory grows linearly

	resetORMTableCachesForTesting()
	runtime.GC()

	measurements := []struct {
		tableCount int
		alloc      uint64
	}{
		{0, getMemStats().Alloc},
	}

	// Compile different table types to simulate real usage
	for i := 1; i <= 10; i++ {
		// Force GC
		runtime.GC()

		// Compile a new table type (alternate between our mock tables)
		switch i % 4 {
		case 1:
			_ = MakeScyllaTable[simpleRecord, simpleSchema]()
		case 2:
			_ = MakeScyllaTable[mediumRecord, mediumSchema]()
		case 3:
			_ = MakeScyllaTable[complexRecord, complexSchema]()
		case 0:
			_ = MakeScyllaTable[veryComplexRecord, veryComplexSchema]()
		}

		runtime.GC()
		alloc := getMemStats().Alloc
		measurements = append(measurements, struct {
			tableCount int
			alloc      uint64
		}{i, alloc})
	}

	t.Log("=== Memory Growth Pattern ===")
	for i, m := range measurements {
		if i == 0 {
			t.Logf("Baseline: %s", formatBytes(m.alloc))
		} else {
			delta := int64(m.alloc) - int64(measurements[i-1].alloc)
			cumulative := int64(m.alloc) - int64(measurements[0].alloc)
			t.Logf("After table %d: %s (delta: %s, cumulative: %s)",
				m.tableCount,
				formatBytes(m.alloc),
				formatBytes(uint64(delta)),
				formatBytes(uint64(cumulative)))
		}
	}
}
