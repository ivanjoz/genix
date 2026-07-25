package types

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
)

type WarehouseStockMin struct {
	WarehouseID int32 `json:"a"`
	Quantity    int32 `json:"c"`
}

type ProductPresentation struct {
	ID              int16  `ms:"i" json:"id,omitempty"`
	AtributoID      int16  `ms:"a" json:"at,omitempty"`
	Name            string `ms:"n" json:"nm,omitempty"`
	Color           string `ms:"c" json:"cl,omitempty"`
	Price           int32  `ms:"p" json:"pc,omitempty"`
	PriceDifference int32  `ms:"d" json:"pd,omitempty"`
	SKU             string `ms:"sk" json:"sk,omitempty"`
	Status          int8   `ms:"s" json:"ss,omitempty"`
}

type Product struct {
	scylla.TableStruct[ProductTable, Product]
	CompanyID     int32   `json:",omitempty"`
	ID            int32   `db:"id,pk"`
	TempID        int32   `json:",omitempty"`
	Name          string  `db:"nombre"`
	Description   string  `json:",omitempty"`
	ContentHTML   string  `json:",omitempty"`
	CategoryIDs   []int32 `json:",omitempty" db:"category_ids"`
	BrandID       int32   `json:",omitempty"`
	Params        []int8  `json:",omitempty"`
	Price         int32   `json:",omitempty"`
	CurrencyID    int16   `json:",omitempty"`
	UnitID        int16   `json:",omitempty"`
	Discount      float32 `json:",omitempty"`
	FinalPrice    int32   `json:",omitempty"`
	Weight        float32 `json:",omitempty"`
	Volume        float32 `json:",omitempty"`
	SbuQuantity   int32   `json:",omitempty"`
	SbuUnit       string  `json:",omitempty"`
	SbuPrice      int32   `json:",omitempty"`
	SbuDiscount   float32 `json:",omitempty"`
	SbuFinalPrice int32   `json:",omitempty"`
	SKU           string  `json:",omitempty"`
	NameHash      int32   `json:",omitempty"`

	Properties    []ProductProperties   `json:",omitempty"`
	Presentations []ProductPresentation `json:",omitempty"`
	// Image storage: ImageMain is the imageID of the primary image (defaults to the
	// first uploaded). ImageIDs holds every imageID; ImageDescriptions is parallel to it.
	// Each imageID encodes its own resolution config in the last digit (autoincrement*10 + configDigit).
	ImageMain         int32               `json:",omitempty"`
	ImageIDs          []int32             `json:",omitempty"`
	ImageDescriptions []string            `json:",omitempty"`
	Stock             []WarehouseStockMin `json:",omitempty"`
	ReservedStock     []WarehouseStockMin `json:",omitempty"`
	StockStatus       int8                `json:",omitempty"`
	NameUpdated       int32               `json:",omitempty"`
	// General properties
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	/* concatenated with CompanyID to be indexed */
	CategoriesWithStock []int32 `json:",omitempty"`
	CacheVersion        uint8   `json:"ccv,omitempty"`
	/* extra */
	BrandName_ string `json:",omitempty"`
}

func (e *Product) GetTextSearchIndex() string {
	name := e.Name
	if len(e.BrandName_) > 0 {
		name += " " + e.BrandName_
	}
	return name
}

type ProductTable struct {
	scylla.TableStruct[ProductTable, Product]
	CompanyID           scylla.Col[ProductTable, int32]
	ID                  scylla.Col[ProductTable, int32]
	Name                scylla.Col[ProductTable, string]
	Description         scylla.Col[ProductTable, string]
	ContentHTML         scylla.Col[ProductTable, string]
	CategoryIDs         scylla.ColSlice[ProductTable, int32] `db:"category_ids"`
	BrandID             scylla.Col[ProductTable, int32]
	Params              scylla.ColSlice[ProductTable, int8] `db:"params_ids"`
	Price               scylla.Col[ProductTable, int32]
	CurrencyID          scylla.Col[ProductTable, int16]
	UnitID              scylla.Col[ProductTable, int16]
	Discount            scylla.Col[ProductTable, float32]
	FinalPrice          scylla.Col[ProductTable, int32]
	Weight              scylla.Col[ProductTable, float32]
	Volume              scylla.Col[ProductTable, float32]
	SbuQuantity         scylla.Col[ProductTable, int32]
	SbuUnit             scylla.Col[ProductTable, string]
	SbuPrice            scylla.Col[ProductTable, int32]
	SbuDiscount         scylla.Col[ProductTable, float32]
	SbuFinalPrice       scylla.Col[ProductTable, int32]
	SKU                 scylla.Col[ProductTable, string]
	NameHash            scylla.Col[ProductTable, int32]
	Properties          scylla.Col[ProductTable, []ProductProperties]
	Presentations       scylla.Col[ProductTable, []ProductPresentation]
	ImageMain           scylla.Col[ProductTable, int32]
	ImageIDs            scylla.ColSlice[ProductTable, int32]
	ImageDescriptions   scylla.ColSlice[ProductTable, string]
	Stock               scylla.Col[ProductTable, []WarehouseStockMin]
	ReservedStock       scylla.Col[ProductTable, []WarehouseStockMin]
	StockStatus         scylla.Col[ProductTable, int8]
	NameUpdated         scylla.Col[ProductTable, int32]
	Status              scylla.Col[ProductTable, int8]
	Updated             scylla.Col[ProductTable, int32]
	UpdatedBy           scylla.Col[ProductTable, int32]
	Created             scylla.Col[ProductTable, int32]
	CreatedBy           scylla.Col[ProductTable, int32]
	CategoriesWithStock scylla.ColSlice[ProductTable, int32]
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

func (e ProductTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:             "products",
		Partition:        e.CompanyID,
		TextSearchColumn: e.Name,
		SaveCacheVersion: true,
		// Label lookups resolve name + SKU + price/brand without shipping the whole product row.
		GenericRecord: scylla.GenericRecordSchema{
			Name: e.Name, S1: e.SKU, N1: e.FinalPrice, N2: e.BrandID,
		},
		Keys: scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeGlobalIndex, Keys: scylla.Cols(e.CategoriesWithStock)},
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.NameUpdated)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.NameHash), Cols: scylla.Cols(e.ID, e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.StockStatus)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}

type ProductProperty struct {
	ID     int16  `json:"id,omitempty" ms:"i"`
	Name   string `json:"nm,omitempty" ms:"n"`
	Status int8   `json:"ss,omitempty" ms:"s"`
}

type ProductProperties struct {
	ID         int16                       `ms:"i"`
	Name       string                      `ms:"n"`
	Options    []ProductProperty           `ms:"o"`
	Status     int8                        `ms:"s"`
	OptionsMap map[string]*ProductProperty `json:"-" ms:"-"`
}

type Warehouse struct {
	scylla.TableStruct[WarehouseTable, Warehouse]
	CompanyID   int32
	ID          int32
	SiteID      int32
	Name        string
	Description string
	Layout      []WarehouseLayout
	// General properties
	Status    int8   `json:"ss,omitempty" db:"status,view"`
	Updated   int32  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32  `json:",omitempty" db:"updated_by"`
	Created   int32  `json:",omitempty" db:"created"`
	CreatedBy int32  `json:",omitempty" db:"created_by"`
	City      string `json:",omitempty"`
}

type WarehouseTable struct {
	scylla.TableStruct[WarehouseTable, Warehouse]
	CompanyID   scylla.Col[WarehouseTable, int32]
	ID          scylla.Col[WarehouseTable, int32]
	SiteID      scylla.Col[WarehouseTable, int32]
	Name        scylla.Col[WarehouseTable, string]
	Description scylla.Col[WarehouseTable, string]
	Layout      scylla.Col[WarehouseTable, []WarehouseLayout]
	Status      scylla.Col[WarehouseTable, int8]
	Updated     scylla.Col[WarehouseTable, int32]
	UpdatedBy   scylla.Col[WarehouseTable, int32]
	Created     scylla.Col[WarehouseTable, int32]
	CreatedBy   scylla.Col[WarehouseTable, int32]
}

func (e WarehouseTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "warehouses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}

type WarehouseLayout struct {
	ID      int16                  `ms:"i" json:"id,omitempty"`
	Name    string                 `ms:"n" json:"nm,omitempty"`
	RowCant int8                   `ms:"r" json:"rc,omitempty"`
	ColCant int8                   `ms:"c" json:"cc,omitempty"`
	Bloques []WarehouseLayoutBlock `ms:"b" json:"bl,omitempty"`
}

type WarehouseLayoutBlock struct {
	Row    int8   `json:"rw" ms:"r"`
	Column int8   `json:"co" ms:"c"`
	Name   string `json:"nm" ms:"n"`
}

type Site struct {
	scylla.TableStruct[SiteTable, Site]
	CompanyID   int32  `db:"empresa_id,pk"`
	ID          int32  `db:"id,pk"`
	Name        string `db:"nombre"`
	Description string `db:"descripcion"`
	Address     string `db:"direccion"`
	CityID      int32  `db:"pais_ciudad_id"`
	City        string `json:",omitempty" db:"-"`
	Status      int8   `json:"ss,omitempty" db:"status,view"`
	Updated     int32  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy   int32  `json:",omitempty" db:"updated_by"`
	Created     int32  `json:",omitempty" db:"created"`
	CreatedBy   int32  `json:",omitempty" db:"created_by"`
}

type SiteTable struct {
	scylla.TableStruct[SiteTable, Site]
	CompanyID   scylla.Col[SiteTable, int32]
	ID          scylla.Col[SiteTable, int32]
	Name        scylla.Col[SiteTable, string]
	Description scylla.Col[SiteTable, string]
	Address     scylla.Col[SiteTable, string]
	CityID      scylla.Col[SiteTable, int32] `db:"pais_ciudad_id"`
	Status      scylla.Col[SiteTable, int8]
	Updated     scylla.Col[SiteTable, int32]
	UpdatedBy   scylla.Col[SiteTable, int32]
	Created     scylla.Col[SiteTable, int32]
	CreatedBy   scylla.Col[SiteTable, int32]
}

func (e SiteTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "sites",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
