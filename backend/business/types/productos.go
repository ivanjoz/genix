package types

import (
	"app/core"
	"app/db"
)

type ProductImage struct {
	Name        string `ms:"n" json:"n" cbor:"n"`
	Description string `ms:"d" json:"d" cbor:"d"`
}

type WarehouseStockMin struct {
	WarehouseID int32 `cbor:"a" json:"a"`
	Quantity    int32 `cbor:"c" json:"c"`
}

type ProductPresentation struct {
	ID             int16  `ms:"i" json:"id,omitempty" cbor:"i"`
	AtributoID     int16  `ms:"a" json:"at,omitempty" cbor:"a"`
	Name           string `ms:"n" json:"nm,omitempty" cbor:"n"`
	Color          string `ms:"c" json:"cl,omitempty" cbor:"c"`
	Price          int32  `ms:"p" json:"pc,omitempty" cbor:"p"`
	PriceDifference int32 `ms:"d" json:"pd,omitempty" cbor:"d"`
	SKU            string `ms:"sk" json:"sk,omitempty" cbor:"sk"`
	Status         int8   `ms:"s" json:"ss,omitempty" cbor:"s"`
}

type Product struct {
	db.TableStruct[ProductTable, Product]
	CompanyID          int32   `json:",omitempty"`
	ID                 int32   `db:"id,pk"`
	TempID             int32   `json:",omitempty"`
	Name               string  `db:"nombre"`
	Description        string  `json:",omitempty"`
	ContentHTML        string  `json:",omitempty"`
	CategoryIDs        []int32 `json:",omitempty" db:"category_ids"`
	BrandID            int32   `json:",omitempty"`
	Params             []int8  `json:",omitempty"`
	Price              int32   `json:",omitempty"`
	CurrencyID         int16   `json:",omitempty"`
	UnitID             int16   `json:",omitempty"`
	Discount           float32 `json:",omitempty"`
	FinalPrice         int32   `json:",omitempty"`
	Weight             float32 `json:",omitempty"`
	Volume             float32 `json:",omitempty"`
	SbuQuantity        int32   `json:",omitempty"`
	SbuUnit            string  `json:",omitempty"`
	SbuPrice           int32   `json:",omitempty"`
	SbuDiscount        float32 `json:",omitempty"`
	SbuFinalPrice      int32   `json:",omitempty"`
	SKU                string  `json:",omitempty"`
	NameHash           int32   `json:",omitempty"`

	Properties     []ProductProperties   `json:",omitempty"`
	Presentations  []ProductPresentation `json:",omitempty"`
	Images         []ProductImage        `json:",omitempty"`
	Stock          []WarehouseStockMin   `json:",omitempty"`
	ReservedStock  []WarehouseStockMin   `json:",omitempty"`
	StockStatus    int8                  `json:",omitempty"`
	NameUpdated    int32                 `json:",omitempty"`
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
	db.TableStruct[ProductTable, Product]
	CompanyID           db.Col[ProductTable, int32]
	ID                  db.Col[ProductTable, int32]
	Name                db.Col[ProductTable, string]
	Description         db.Col[ProductTable, string]
	ContentHTML         db.Col[ProductTable, string]
	CategoryIDs         db.ColSlice[ProductTable, int32]  `db:"category_ids"`
	BrandID             db.Col[ProductTable, int32]
	Params              db.ColSlice[ProductTable, int8]   `db:"params_ids"`
	Price               db.Col[ProductTable, int32]
	CurrencyID          db.Col[ProductTable, int16]
	UnitID              db.Col[ProductTable, int16]
	Discount            db.Col[ProductTable, float32]
	FinalPrice          db.Col[ProductTable, int32]
	Weight              db.Col[ProductTable, float32]
	Volume              db.Col[ProductTable, float32]
	SbuQuantity         db.Col[ProductTable, int32]
	SbuUnit             db.Col[ProductTable, string]
	SbuPrice            db.Col[ProductTable, int32]
	SbuDiscount         db.Col[ProductTable, float32]
	SbuFinalPrice       db.Col[ProductTable, int32]
	SKU                 db.Col[ProductTable, string]
	NameHash            db.Col[ProductTable, int32]
	Properties          db.Col[ProductTable, []ProductProperties]
	Presentations       db.Col[ProductTable, []ProductPresentation]
	Images              db.Col[ProductTable, []ProductImage]
	Stock               db.Col[ProductTable, []WarehouseStockMin]
	ReservedStock       db.Col[ProductTable, []WarehouseStockMin]
	StockStatus         db.Col[ProductTable, int8]
	NameUpdated         db.Col[ProductTable, int32]
	Status              db.Col[ProductTable, int8]
	Updated             db.Col[ProductTable, int32]
	UpdatedBy           db.Col[ProductTable, int32]
	Created             db.Col[ProductTable, int32]
	CreatedBy           db.Col[ProductTable, int32]
	CategoriesWithStock db.ColSlice[ProductTable, int32]
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
		Name:             "products",
		Partition:        e.CompanyID,
		TextSearchColumn: e.Name,
		SaveCacheVersion: true,
		Keys:             []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeGlobalIndex, Keys: []db.Coln{e.CategoriesWithStock}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.NameUpdated}},
			{Type: db.TypeView, Keys: []db.Coln{e.NameHash}, Cols: []db.Coln{e.ID, e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.StockStatus}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ProductProperty struct {
	ID     int16  `json:"id,omitempty" ms:"i" cbor:"i"`
	Name   string `json:"nm,omitempty" ms:"n" cbor:"n"`
	Status int8   `json:"ss,omitempty" ms:"s" cbor:"s"`
}

type ProductProperties struct {
	ID         int16                       `ms:"i" cbor:"i"`
	Name       string                      `ms:"n" cbor:"n"`
	Options    []ProductProperty           `ms:"o" cbor:"o"`
	Status     int8                        `ms:"s" cbor:"s"`
	OptionsMap map[string]*ProductProperty `json:"-" ms:"-"`
}

type Warehouse struct {
	db.TableStruct[WarehouseTable, Warehouse]
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
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   db.Col[WarehouseTable, int32]
	ID          db.Col[WarehouseTable, int32]
	SiteID      db.Col[WarehouseTable, int32]
	Name        db.Col[WarehouseTable, string]
	Description db.Col[WarehouseTable, string]
	Layout      db.Col[WarehouseTable, []WarehouseLayout]
	Status      db.Col[WarehouseTable, int8]
	Updated     db.Col[WarehouseTable, int32]
	UpdatedBy   db.Col[WarehouseTable, int32]
	Created     db.Col[WarehouseTable, int32]
	CreatedBy   db.Col[WarehouseTable, int32]
}

func (e WarehouseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "warehouses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type WarehouseLayout struct {
	ID      int16                  `ms:"i" cbor:"i" json:"id,omitempty"`
	Name    string                 `ms:"n" cbor:"n" json:"nm,omitempty"`
	RowCant int8                   `ms:"r" cbor:"r" json:"rc,omitempty"`
	ColCant int8                   `ms:"c" cbor:"c" json:"cc,omitempty"`
	Bloques []WarehouseLayoutBlock `ms:"b" cbor:"b" json:"bl,omitempty"`
}

type WarehouseLayoutBlock struct {
	Row    int8   `json:"rw" ms:"r" cbor:"r"`
	Column int8   `json:"co" ms:"c" cbor:"c"`
	Name   string `json:"nm" ms:"n" cbor:"n"`
}

type Site struct {
	db.TableStruct[SiteTable, Site]
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
	db.TableStruct[SiteTable, Site]
	CompanyID   db.Col[SiteTable, int32]
	ID          db.Col[SiteTable, int32]
	Name        db.Col[SiteTable, string]
	Description db.Col[SiteTable, string]
	Address     db.Col[SiteTable, string]
	CityID      db.Col[SiteTable, int32] `db:"pais_ciudad_id"`
	Status      db.Col[SiteTable, int8]
	Updated     db.Col[SiteTable, int32]
	UpdatedBy   db.Col[SiteTable, int32]
	Created     db.Col[SiteTable, int32]
	CreatedBy   db.Col[SiteTable, int32]
}

func (e SiteTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sites",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
