// Package types holds the production catalog contracts: the Product row and everything
// hanging off it. It lives here rather than in app/business because the catalog is the root
// of the Producción module — recipes, production orders and per-unit costing all key off
// Product, and a supply/material is the same row at Status = 2.
package types

import (
	"app/core"
	"app/db"
)

type WarehouseStockMin struct {
	WarehouseID int32 `json:"a" cb:"a,1"`
	Quantity    int32 `json:"c" cb:"c,2"`
}

type ProductPresentation struct {
	ID              int16  `ms:"i" json:"id,omitempty" cb:"id,1"`
	AtributoID      int16  `ms:"a" json:"at,omitempty" cb:"at,2"`
	Name            string `ms:"n" json:"nm,omitempty" cb:"nm,3"`
	Color           string `ms:"c" json:"cl,omitempty" cb:"cl,4"`
	Price           int32  `ms:"p" json:"pc,omitempty" cb:"pc,5"`
	PriceDifference int32  `ms:"d" json:"pd,omitempty" cb:"pd,6"`
	SKU             string `ms:"sk" json:"sk,omitempty" cb:"sk,7"`
	Status          int8   `ms:"s" json:"ss,omitempty" cb:"ss,8"`
}

// Product.Status is the row's role in the catalog, and it is enumerated in the delta view's
// FixedValues below — so these three are the only legal values, and adding one means changing
// the schema. A supply/material shares the catalog, the stock engine and the movement ledger
// with an ordinary product; the status is the only thing that keeps it out of every
// product-facing flow, since those all pin Active.
//
// An asset is a supply whose DepreciationMonths > 0. Nothing else distinguishes it here — the
// accounting overlay lives in app/accounting.
const (
	ProductStatusInactive int8 = 0 // Evicted / soft-deleted; delta syncs ship these as removals.
	ProductStatusActive   int8 = 1
	ProductStatusSupply   int8 = 2
)

type Product struct {
	db.TableStruct[ProductTable, Product]
	CompanyID   int32   `json:",omitempty" cb:"CompanyID,1"`
	ID          int32   `db:"id,pk" cb:"ID,2"`
	TempID      int32   `json:",omitempty" cb:"TempID,3"`
	Name        string  `db:"nombre" cb:"Name,4"`
	Description string  `json:",omitempty" cb:"Description,5"`
	ContentHTML string  `json:",omitempty" cb:"ContentHTML,6"`
	CategoryIDs []int32 `json:",omitempty" db:"category_ids" cb:"CategoryIDs,7"`
	BrandID     int32   `json:",omitempty" cb:"BrandID,8"`
	Params      []int8  `json:",omitempty" cb:"Params,9"`
	Price       int32   `json:",omitempty" cb:"Price,10"`
	CurrencyID  int16   `json:",omitempty" cb:"CurrencyID,11"`
	UnitID      int16   `json:",omitempty" cb:"UnitID,12"`
	Discount    float32 `json:",omitempty" cb:"Discount,13"`
	FinalPrice  int32   `json:",omitempty" cb:"FinalPrice,14"`
	Weight      float32 `json:",omitempty" cb:"Weight,15"`
	Volume      float32 `json:",omitempty" cb:"Volume,16"`
	// Sub-unit: the divisor of the (Units, Sub) quantity pair. SbuQuantity is how many
	// sub-units make one whole unit — 1000 for a kilogram sold by the gram, 6 for a box sold
	// by the candy. 0 or 1 means the product has no sub-unit. It may only ever be refined to
	// a multiple (see core.IsQuantityDivisorRefinement): coarsening would round away real
	// stock, since 9/12 is 4.5 sixths. int16 to match SubDivisor on the stock and movement
	// rows, so one concept has one type.
	SbuQuantity   int16   `json:",omitempty" cb:"SbuQuantity,17"`
	SbuUnit       string  `json:",omitempty" cb:"SbuUnit,18"`
	SbuPrice      int32   `json:",omitempty" cb:"SbuPrice,19"`
	SbuDiscount   float32 `json:",omitempty" cb:"SbuDiscount,20"`
	SbuFinalPrice int32   `json:",omitempty" cb:"SbuFinalPrice,21"`
	SKU           string  `json:",omitempty" cb:"SKU,22"`
	NameHash      int32   `json:",omitempty" cb:"NameHash,23"`
	// Straight-line depreciation term for supplies that are fixed assets (Status=2).
	// 0 = not depreciable. Copied onto accounting.Asset at acquisition so later catalog
	// edits never rewrite the schedule of an asset already in service.
	DepreciationMonths int16 `json:",omitempty" cb:"DepreciationMonths,24"`

	Properties    []ProductProperties   `json:",omitempty" cb:"Properties,25"`
	Presentations []ProductPresentation `json:",omitempty" cb:"Presentations,26"`
	// Image storage: ImageMain is the imageID of the primary image (defaults to the
	// first uploaded). ImageIDs holds every imageID; ImageDescriptions is parallel to it.
	// Each imageID encodes its own resolution config in the last digit (autoincrement*10 + configDigit).
	ImageMain         int32               `json:",omitempty" cb:"ImageMain,27"`
	ImageIDs          []int32             `json:",omitempty" cb:"ImageIDs,28"`
	ImageDescriptions []string            `json:",omitempty" cb:"ImageDescriptions,29"`
	Stock             []WarehouseStockMin `json:",omitempty" cb:"Stock,30"`
	ReservedStock     []WarehouseStockMin `json:",omitempty" cb:"ReservedStock,31"`
	StockStatus       int8                `json:",omitempty" cb:"StockStatus,32"`
	NameUpdated       int32               `json:",omitempty" cb:"NameUpdated,33"`
	// General properties
	Status    int8  `json:"ss,omitempty" cb:"ss,34"`
	Updated   int32 `json:"upd,omitempty" cb:"upd,35"`
	UpdatedBy int32 `json:",omitempty" cb:"UpdatedBy,36"`
	Created   int32 `json:",omitempty" cb:"Created,37"`
	CreatedBy int32 `json:",omitempty" cb:"CreatedBy,38"`
	/* concatenated with CompanyID to be indexed */
	CategoriesWithStock []int32 `json:",omitempty" cb:"CategoriesWithStock,39"`
	Version             int16   `json:"ver,omitempty" cb:"ver,40"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt,omitempty" cb:"mdt,41"`
	// BrandName copies the brand's name so the text index can read it from the table: a reindex
	// only sees stored columns. PostProducts fills it from BrandID.
	BrandName string `json:",omitempty" cb:"BrandName,42"`
}

type ProductTable struct {
	db.TableStruct[ProductTable, Product]
	CompanyID           db.Col[*ProductTable, int32]
	ID                  db.Col[*ProductTable, int32]
	Name                db.Col[*ProductTable, string]
	Description         db.Col[*ProductTable, string]
	ContentHTML         db.Col[*ProductTable, string]
	CategoryIDs         db.ColSlice[*ProductTable, int32] `db:"category_ids"`
	BrandID             db.Col[*ProductTable, int32]
	BrandName           db.Col[*ProductTable, string]
	Params              db.ColSlice[*ProductTable, int8] `db:"params_ids"`
	Price               db.Col[*ProductTable, int32]
	CurrencyID          db.Col[*ProductTable, int16]
	UnitID              db.Col[*ProductTable, int16]
	Discount            db.Col[*ProductTable, float32]
	FinalPrice          db.Col[*ProductTable, int32]
	Weight              db.Col[*ProductTable, float32]
	Volume              db.Col[*ProductTable, float32]
	SbuQuantity         db.Col[*ProductTable, int16]
	SbuUnit             db.Col[*ProductTable, string]
	SbuPrice            db.Col[*ProductTable, int32]
	SbuDiscount         db.Col[*ProductTable, float32]
	SbuFinalPrice       db.Col[*ProductTable, int32]
	SKU                 db.Col[*ProductTable, string]
	NameHash            db.Col[*ProductTable, int32]
	DepreciationMonths  db.Col[*ProductTable, int16]
	Properties          db.Col[*ProductTable, []ProductProperties]
	Presentations       db.Col[*ProductTable, []ProductPresentation]
	ImageMain           db.Col[*ProductTable, int32]
	ImageIDs            db.ColSlice[*ProductTable, int32]
	ImageDescriptions   db.ColSlice[*ProductTable, string]
	Stock               db.Col[*ProductTable, []WarehouseStockMin]
	ReservedStock       db.Col[*ProductTable, []WarehouseStockMin]
	StockStatus         db.Col[*ProductTable, int8]
	NameUpdated         db.Col[*ProductTable, int32]
	Status              db.Col[*ProductTable, int8]
	Updated             db.Col[*ProductTable, int32]
	UpdatedBy           db.Col[*ProductTable, int32]
	Created             db.Col[*ProductTable, int32]
	CreatedBy           db.Col[*ProductTable, int32]
	CategoriesWithStock db.ColSlice[*ProductTable, int32]
	Version             db.Col[*ProductTable, int16]
}

func (e *Product) FillCategoriesWithStock() {
	e.CategoriesWithStock = nil
	if e.StockStatus > 0 {
		for _, cid := range e.CategoryIDs {
			e.CategoriesWithStock = append(e.CategoriesWithStock, e.CompanyID*10000+cid)
		}
	}
}

func (e *Product) SelfParse() {
	e.NameHash = core.BasicHashInt(core.NormalizeString(&e.Name))
}

func (e ProductTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        22,
		Name:      "products",
		Partition: e.CompanyID,
		// Only active products (Status != 0) are searchable.
		TextSearch: &db.TextSearch{Columns: db.Cols(e.Name, e.BrandName), Status: e.Status},
		CacheByIDs: true,
		// Label lookups resolve name + SKU + price/brand without shipping the whole product row.
		GenericRecord: db.GenericRecordSchema{
			Name: e.Name, S1: e.SKU, N1: e.FinalPrice, N2: e.BrandID,
		},
		Keys: db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		// 2 = supply/material: same catalog row, same stock engine, but excluded from every
		// product flow because they all pin Status=1 (products.go, product-ecommerce.go).
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1, 2}},
		},
		Indexes: []db.Index{
			{Type: db.TypeGlobalIndex, Keys: db.Cols(e.CategoriesWithStock)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.NameUpdated)},
			{Type: db.TypeView, Keys: db.Cols(e.NameHash), Cols: db.Cols(e.ID, e.Status)},
			{Type: db.TypeView, Keys: db.Cols(e.StockStatus)},
			// Still read by the ecommerce delta (product-ecommerce.go), which keeps its timestamp
			// watermark because it also drives the prerendered .db snapshot.
			{Type: db.TypeView, Keys: db.Cols(e.Status)},
			{Type: db.TypeView, Keys: db.Cols(e.Updated)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

type ProductProperty struct {
	ID     int16  `json:"id,omitempty" ms:"i" cb:"id,1"`
	Name   string `json:"nm,omitempty" ms:"n" cb:"nm,2"`
	Status int8   `json:"ss,omitempty" ms:"s" cb:"ss,3"`
}

type ProductProperties struct {
	ID      int16             `ms:"i"`
	Name    string            `ms:"n"`
	Options []ProductProperty `ms:"o"`
	Status  int8              `ms:"s"`
}
