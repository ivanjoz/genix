package db

import (
	"testing"
)

// Table declaring a full GenericRecord mapping across mixed integer widths, so the precompiled
// accessors are exercised for int16/int32/int64 and for the auto-resolved status column.
type genericLabelRecord struct {
	TableStruct[genericLabelSchema, genericLabelRecord]
	CompanyID    int32  `db:"company_id"`
	ID           int32  `db:"id"`
	Name         string `db:"name"`
	SKU          string `db:"sku"`
	City         string `db:"city"`
	Price        int64  `db:"price"`
	BrandID      int16  `db:"brand_id"`
	Status       int8   `db:"status"`
	CacheVersion uint8  `json:"ccv,omitempty"`
}

type genericLabelSchema struct {
	TableStruct[genericLabelSchema, genericLabelRecord]
	CompanyID Col[genericLabelSchema, int32]
	ID        Col[genericLabelSchema, int32]
	Name      Col[genericLabelSchema, string]
	SKU       Col[genericLabelSchema, string]
	City      Col[genericLabelSchema, string]
	Price     Col[genericLabelSchema, int64]
	BrandID   Col[genericLabelSchema, int16]
	Status    Col[genericLabelSchema, int8]
}

func (e genericLabelSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:             "generic_label",
		Partition:        e.CompanyID,
		SaveCacheVersion: true,
		GenericRecord: GenericRecordSchema{
			Name: e.Name, S1: e.SKU, S2: e.City, N1: e.Price, N2: e.BrandID,
		},
		Keys: Cols(e.ID.Autoincrement(0)),
	}
}

// Table with SaveCacheVersion but no GenericRecord: must stay unexposed.
type genericOptedOutRecord struct {
	TableStruct[genericOptedOutSchema, genericOptedOutRecord]
	CompanyID    int32  `db:"company_id"`
	ID           int32  `db:"id"`
	Name         string `db:"name"`
	CacheVersion uint8  `json:"ccv,omitempty"`
}

type genericOptedOutSchema struct {
	TableStruct[genericOptedOutSchema, genericOptedOutRecord]
	CompanyID Col[genericOptedOutSchema, int32]
	ID        Col[genericOptedOutSchema, int32]
	Name      Col[genericOptedOutSchema, string]
}

func (e genericOptedOutSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:             "generic_opted_out",
		Partition:        e.CompanyID,
		SaveCacheVersion: true,
		Keys:             Cols(e.ID.Autoincrement(0)),
	}
}

// GenericRecord without SaveCacheVersion must panic: ccv is what makes the read incremental.
type genericNoCacheVersionRecord struct {
	TableStruct[genericNoCacheVersionSchema, genericNoCacheVersionRecord]
	CompanyID int32  `db:"company_id"`
	ID        int32  `db:"id"`
	Name      string `db:"name"`
}

type genericNoCacheVersionSchema struct {
	TableStruct[genericNoCacheVersionSchema, genericNoCacheVersionRecord]
	CompanyID Col[genericNoCacheVersionSchema, int32]
	ID        Col[genericNoCacheVersionSchema, int32]
	Name      Col[genericNoCacheVersionSchema, string]
}

func (e genericNoCacheVersionSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:          "generic_no_cache_version",
		Partition:     e.CompanyID,
		GenericRecord: GenericRecordSchema{Name: e.Name},
		Keys:          Cols(e.ID.Autoincrement(0)),
	}
}

// A non-string column in a string slot must panic rather than mis-scan at runtime.
type genericBadNameRecord struct {
	TableStruct[genericBadNameSchema, genericBadNameRecord]
	CompanyID    int32 `db:"company_id"`
	ID           int32 `db:"id"`
	Amount       int32 `db:"amount"`
	CacheVersion uint8 `json:"ccv,omitempty"`
}

type genericBadNameSchema struct {
	TableStruct[genericBadNameSchema, genericBadNameRecord]
	CompanyID Col[genericBadNameSchema, int32]
	ID        Col[genericBadNameSchema, int32]
	Amount    Col[genericBadNameSchema, int32]
}

func (e genericBadNameSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:             "generic_bad_name",
		Partition:        e.CompanyID,
		SaveCacheVersion: true,
		GenericRecord:    GenericRecordSchema{Name: e.Amount},
		Keys:             Cols(e.ID.Autoincrement(0)),
	}
}

func TestGenericRecordPlanProjectionOrder(t *testing.T) {
	scyllaTable := MakeScyllaTable[genericLabelRecord, genericLabelSchema]()

	plan := scyllaTable.genericRecordPlan
	if plan == nil {
		t.Fatal("expected a precompiled genericRecordPlan for a table declaring GenericRecord")
	}

	// The projection is built once at table-build time; ID leads and status trails.
	expectedProjection := "id, name, sku, city, price, brand_id, status"
	if plan.projection != expectedProjection {
		t.Fatalf("projection mismatch\n got: %v\nwant: %v", plan.projection, expectedProjection)
	}
	if len(plan.accessors) != 7 {
		t.Fatalf("expected 7 precompiled accessors, got %v", len(plan.accessors))
	}
}

// The precompiled accessors must widen every declared integer width and write into the right field.
func TestGenericRecordAccessorsAssignEveryField(t *testing.T) {
	scyllaTable := MakeScyllaTable[genericLabelRecord, genericLabelSchema]()
	scanTargets, scanSlots := scyllaTable.genericRecordPlan.newScanSlots()

	// Emulate what gocql does: write into the typed buffers the plan handed out.
	*(scanTargets[0].(*int32)) = 4210
	*(scanTargets[1].(*string)) = "Café molido"
	*(scanTargets[2].(*string)) = "SKU-77"
	*(scanTargets[3].(*string)) = "Lima"
	*(scanTargets[4].(*int64)) = 1899
	*(scanTargets[5].(*int16)) = 12
	*(scanTargets[6].(*int8)) = 1

	record := GenericRecord{}
	for _, scanSlot := range scanSlots {
		scanSlot.assign(&record)
	}

	expected := GenericRecord{ID: 4210, Name: "Café molido", S1: "SKU-77", S2: "Lima", N1: 1899, N2: 12, Status: 1}
	if record != expected {
		t.Fatalf("assigned record mismatch\n got: %+v\nwant: %+v", record, expected)
	}
}

// Buffers are reused across rows, so a second scan must fully overwrite the previous values.
func TestGenericRecordScanSlotsAreReusedAcrossRows(t *testing.T) {
	scyllaTable := MakeScyllaTable[genericLabelRecord, genericLabelSchema]()
	scanTargets, scanSlots := scyllaTable.genericRecordPlan.newScanSlots()

	assignRow := func() GenericRecord {
		record := GenericRecord{}
		for _, scanSlot := range scanSlots {
			scanSlot.assign(&record)
		}
		return record
	}

	*(scanTargets[0].(*int32)) = 1
	*(scanTargets[1].(*string)) = "first"
	firstRecord := assignRow()

	*(scanTargets[0].(*int32)) = 2
	*(scanTargets[1].(*string)) = "second"
	secondRecord := assignRow()

	if firstRecord.ID != 1 || firstRecord.Name != "first" {
		t.Fatalf("first row was mutated by the second scan: %+v", firstRecord)
	}
	if secondRecord.ID != 2 || secondRecord.Name != "second" {
		t.Fatalf("second row did not pick up the new buffer values: %+v", secondRecord)
	}
}

func TestGenericRecordPlanAbsentWhenNotDeclared(t *testing.T) {
	scyllaTable := MakeScyllaTable[genericOptedOutRecord, genericOptedOutSchema]()
	if scyllaTable.genericRecordPlan != nil {
		t.Fatal("a table without GenericRecord must not be exposed through the generic query")
	}
}

func TestQueryCachedGenericByIDsRejectsUnregisteredAndOptedOutTables(t *testing.T) {
	if _, err := QueryCachedGenericByIDs("table_that_does_not_exist", []IDCacheVersion{{ID: 1}}); err == nil {
		t.Fatal("expected an error for a table name that was never registered")
	}

	RegisterTableFactory("generic_opted_out", func() ScyllaTable {
		return MakeScyllaTable[genericOptedOutRecord, genericOptedOutSchema]()
	})
	if _, err := QueryCachedGenericByIDs("generic_opted_out", []IDCacheVersion{{ID: 1}}); err == nil {
		t.Fatal("expected an error for a registered table that did not declare GenericRecord")
	}
}

// An empty request must short-circuit before any table or registry lookup.
func TestQueryCachedGenericByIDsIgnoresEmptyRequest(t *testing.T) {
	records, err := QueryCachedGenericByIDs("generic_label", nil)
	if err != nil || records != nil {
		t.Fatalf("expected a no-op for an empty request, got records=%v err=%v", records, err)
	}
}

func TestGenericRecordSchemaValidationPanics(t *testing.T) {
	testCases := []struct {
		name       string
		makeTable  func()
		wantReason string
	}{
		{
			name:       "GenericRecord without SaveCacheVersion",
			makeTable:  func() { MakeScyllaTable[genericNoCacheVersionRecord, genericNoCacheVersionSchema]() },
			wantReason: "requires SaveCacheVersion enabled",
		},
		{
			name:       "non-string column in the Name slot",
			makeTable:  func() { MakeScyllaTable[genericBadNameRecord, genericBadNameSchema]() },
			wantReason: "must be string",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatalf("expected a panic mentioning %q", testCase.wantReason)
				}
			}()
			testCase.makeTable()
		})
	}
}
