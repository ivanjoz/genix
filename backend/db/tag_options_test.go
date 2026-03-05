package db

import "testing"

type collectionDefaultColRecord struct {
	TableStruct[collectionDefaultColSchema, collectionDefaultColRecord]
	EmpresaID int32   `db:"empresa_id"`
	ID        int64   `db:"id"`
	Items     []int32 `db:"items"`
}

type collectionDefaultColSchema struct {
	TableStruct[collectionDefaultColSchema, collectionDefaultColRecord]
	EmpresaID Col[collectionDefaultColSchema, int32]
	ID        Col[collectionDefaultColSchema, int64]
	Items     Col[collectionDefaultColSchema, []int32]
}

func (e collectionDefaultColSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "collection_default_col",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

type collectionDefaultColSliceSchema struct {
	TableStruct[collectionDefaultColSliceSchema, collectionDefaultColSliceRecord]
	EmpresaID Col[collectionDefaultColSliceSchema, int32]
	ID        Col[collectionDefaultColSliceSchema, int64]
	Items     ColSlice[collectionDefaultColSliceSchema, int32]
}

type collectionDefaultColSliceRecord struct {
	TableStruct[collectionDefaultColSliceSchema, collectionDefaultColSliceRecord]
	EmpresaID int32   `db:"empresa_id"`
	ID        int64   `db:"id"`
	Items     []int32 `db:"items"`
}

func (e collectionDefaultColSliceSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "collection_default_colslice",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

type collectionTagFrozenRecord struct {
	TableStruct[collectionTagFrozenSchema, collectionTagFrozenRecord]
	EmpresaID int32   `db:"empresa_id"`
	ID        int64   `db:"id"`
	Items     []int32 `db:"items,frozen"`
}

type collectionTagFrozenSchema struct {
	TableStruct[collectionTagFrozenSchema, collectionTagFrozenRecord]
	EmpresaID Col[collectionTagFrozenSchema, int32]
	ID        Col[collectionTagFrozenSchema, int64]
	Items     Col[collectionTagFrozenSchema, []int32]
}

func (e collectionTagFrozenSchema) GetSchema() TableSchema {
	return TableSchema{
		Name:      "collection_tag_frozen",
		Partition: e.EmpresaID,
		Keys:      []Coln{e.ID},
	}
}

func TestSliceDefaultsAreSetTypes(t *testing.T) {
	// Validate default collection mapping remains set-backed before Col/ColSlice table-shape defaults are applied.
	testCases := []struct {
		fieldType string
		expected  string
	}{
		{fieldType: "[]string", expected: "set<text>"},
		{fieldType: "[]int32", expected: "set<int>"},
		{fieldType: "*[]int32", expected: "set<int>"},
	}

	for _, testCase := range testCases {
		resolvedType := GetColTypeByName(testCase.fieldType, "")
		if resolvedType.ColType != testCase.expected {
			t.Fatalf("fieldType=%s expected=%s got=%s", testCase.fieldType, testCase.expected, resolvedType.ColType)
		}
	}
}

func TestApplyCollectionTagOptions(t *testing.T) {
	// Confirm tag options remap defaults to set/list/frozen variants deterministically.
	baseType := GetColTypeByName("[]int32", "")

	tagSet := parseDBTagConfig(",set")
	if got := applyCollectionTagOptions("SaleOrder", "DetailPrices", baseType, tagSet).ColType; got != "set<int>" {
		t.Fatalf("expected set<int> got=%s", got)
	}

	tagFrozenList := parseDBTagConfig(",frozen")
	if got := applyCollectionTagOptions("SaleOrder", "DetailPrices", baseType, tagFrozenList).ColType; got != "frozen<list<int>>" {
		t.Fatalf("expected frozen<list<int>> got=%s", got)
	}

	tagFrozenSet := parseDBTagConfig(",frozen,set")
	if got := applyCollectionTagOptions("SaleOrder", "DetailPrices", baseType, tagFrozenSet).ColType; got != "frozen<set<int>>" {
		t.Fatalf("expected frozen<set<int>> got=%s", got)
	}
}

func TestCollectionLiteralBrackets(t *testing.T) {
	// Keep generated literal syntax aligned with the final collection kind used by each column.
	openBracket, closeBracket := getCollectionLiteralBrackets("list<int>")
	if openBracket != "[" || closeBracket != "]" {
		t.Fatalf("expected [] for list<int>, got %s%s", openBracket, closeBracket)
	}

	openBracket, closeBracket = getCollectionLiteralBrackets("frozen<list<int>>")
	if openBracket != "[" || closeBracket != "]" {
		t.Fatalf("expected [] for frozen<list<int>>, got %s%s", openBracket, closeBracket)
	}

	openBracket, closeBracket = getCollectionLiteralBrackets("set<int>")
	if openBracket != "{" || closeBracket != "}" {
		t.Fatalf("expected {} for set<int>, got %s%s", openBracket, closeBracket)
	}
}

func TestCollectionDefaultByTableColumnType(t *testing.T) {
	// Col should freeze collection columns by default, while ColSlice keeps them non-frozen.
	resetORMTableCachesForTesting()

	colSchema := initStructTable[collectionDefaultColSchema, collectionDefaultColRecord](new(collectionDefaultColSchema))
	if got := colSchema.Items.GetInfo().ColType; got != "frozen<set<int>>" {
		t.Fatalf("expected frozen<set<int>> for Col slice mapping, got=%s", got)
	}

	colSliceSchema := initStructTable[collectionDefaultColSliceSchema, collectionDefaultColSliceRecord](new(collectionDefaultColSliceSchema))
	if got := colSliceSchema.Items.GetInfo().ColType; got != "set<int>" {
		t.Fatalf("expected set<int> for ColSlice mapping, got=%s", got)
	}
}

func TestCollectionTagOverridesColDefaultFrozenPolicy(t *testing.T) {
	// Explicit tag options must remain authoritative and not be overridden by Col default freezing rules.
	resetORMTableCachesForTesting()

	colSchema := initStructTable[collectionTagFrozenSchema, collectionTagFrozenRecord](new(collectionTagFrozenSchema))
	if got := colSchema.Items.GetInfo().ColType; got != "frozen<list<int>>" {
		t.Fatalf("expected frozen<list<int>> from db tag override, got=%s", got)
	}
}
