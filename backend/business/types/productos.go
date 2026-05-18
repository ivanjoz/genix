package types

import (
	"app/core"
	"app/db"
)

type ProductImage struct {
	Name        string `ms:"n" json:"n" cbor:"n"`
	Descripcion string `ms:"d" json:"d" cbor:"d"`
}

type WarehouseStockMin struct {
	WarehouseID int32 `cbor:"a" json:"a"`
	Cantidad    int32 `cbor:"c" json:"c"`
}

type ProductPresentation struct {
	ID               int16  `ms:"i" json:"id,omitempty" cbor:"i"`
	AtributoID       int16  `ms:"a" json:"at,omitempty" cbor:"a"`
	Name             string `ms:"n" json:"nm,omitempty" cbor:"n"`
	Color            string `ms:"c" json:"cl,omitempty" cbor:"c"`
	Precio           int32  `ms:"p" json:"pc,omitempty" cbor:"p"`
	DiferenciaPrecio int32  `ms:"d" json:"pd,omitempty" cbor:"d"`
	SKU              string `ms:"sk" json:"sk,omitempty" cbor:"sk"`
	Status           int8   `ms:"s" json:"ss,omitempty" cbor:"s"`
}

type Product struct {
	db.TableStruct[ProductTable, Product]
	CompanyID      int32   `json:",omitempty"`
	ID             int32   `db:"id,pk"`
	TempID         int32   `json:",omitempty"`
	Nombre         string  `db:"nombre"`
	Descripcion    string  `json:",omitempty"`
	ContentHTML    string  `json:",omitempty"`
	CategoriasIDs  []int32 `json:",omitempty"`
	MarcaID        int32   `json:",omitempty"`
	Params         []int8  `json:",omitempty"`
	Precio         int32   `json:",omitempty"`
	MonedaID       int16   `json:",omitempty"`
	UnidadID       int16   `json:",omitempty"`
	Descuento      float32 `json:",omitempty"`
	PrecioFinal    int32   `json:",omitempty"`
	Peso           float32 `json:",omitempty"`
	Volumen        float32 `json:",omitempty"`
	SbnCantidad    int32   `json:",omitempty"`
	SbnUnidad      string  `json:",omitempty"`
	SbnPrecio      int32   `json:",omitempty"`
	SbnDescuento   float32 `json:",omitempty"`
	SbnPrecioFinal int32   `json:",omitempty"`
	SKU            string  `json:",omitempty"`
	NombreHash     int32   `json:",omitempty"`

	Propiedades    []ProductProperties `json:",omitempty"`
	Presentaciones []ProductPresentation `json:",omitempty"`
	Images         []ProductImage      `json:",omitempty"`
	Stock          []WarehouseStockMin     `json:",omitempty"`
	StockReservado []WarehouseStockMin     `json:",omitempty"`
	StockStatus    int8                  `json:",omitempty"`
	NameUpdated    int32                 `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	/* concatenada con la Company-ID para ser indexadas*/
	CategoriasConStock []int32 `json:",omitempty"`
	CacheVersion       uint8   `json:"ccv,omitempty"`
}

type ProductTable struct {
	db.TableStruct[ProductTable, Product]
	CompanyID          db.Col[ProductTable, int32]
	ID                 db.Col[ProductTable, int32]
	Nombre             db.Col[ProductTable, string]
	Descripcion        db.Col[ProductTable, string]
	ContentHTML        db.Col[ProductTable, string]
	CategoriasIDs      db.ColSlice[ProductTable, int32] `db:"categorias_ids"`
	MarcaID            db.Col[ProductTable, int32]
	Params             db.ColSlice[ProductTable, int8] `db:"params_ids"`
	Precio             db.Col[ProductTable, int32]
	MonedaID           db.Col[ProductTable, int16]
	UnidadID           db.Col[ProductTable, int16]
	Descuento          db.Col[ProductTable, float32]
	PrecioFinal        db.Col[ProductTable, int32]
	Peso               db.Col[ProductTable, float32]
	Volumen            db.Col[ProductTable, float32]
	SbnCantidad        db.Col[ProductTable, int32]
	SbnUnidad          db.Col[ProductTable, string]
	SbnPrecio          db.Col[ProductTable, int32]
	SbnDescuento       db.Col[ProductTable, float32]
	SbnPrecioFinal     db.Col[ProductTable, int32]
	SKU                db.Col[ProductTable, string]
	NombreHash         db.Col[ProductTable, int32]
	Propiedades        db.Col[ProductTable, []ProductProperties]
	Presentaciones     db.Col[ProductTable, []ProductPresentation]
	Images             db.Col[ProductTable, []ProductImage]
	Stock              db.Col[ProductTable, []WarehouseStockMin]
	StockReservado     db.Col[ProductTable, []WarehouseStockMin]
	StockStatus        db.Col[ProductTable, int8]
	NameUpdated        db.Col[ProductTable, int32]
	Status             db.Col[ProductTable, int8]
	Updated            db.Col[ProductTable, int32]
	UpdatedBy          db.Col[ProductTable, int32]
	Created            db.Col[ProductTable, int32]
	CreatedBy          db.Col[ProductTable, int32]
	CategoriasConStock db.ColSlice[ProductTable, int32]
}

func (e *Product) FillCategoriasConStock() {
	e.CategoriasConStock = nil
	if e.StockStatus > 0 {
		for _, cid := range e.CategoriasIDs {
			e.CategoriasConStock = append(e.CategoriasConStock, e.CompanyID*10000+cid)
		}
	}
}

func (e *Product) SelfParse() {
	e.NombreHash = core.BasicHashInt(core.NormalizeString(&e.Nombre))
}

func (e ProductTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "productos",
		Partition:        e.CompanyID,
		SaveCacheVersion: true,
		Keys:             []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeGlobalIndex, Keys: []db.Coln{e.CategoriasConStock}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.NameUpdated}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.NombreHash}},
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.StockStatus}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ProductProperty struct {
	ID     int16  `json:"id,omitempty" ms:"i" cbor:"i"`
	Nombre string `json:"nm,omitempty" ms:"n" cbor:"n"`
	Status int8   `json:"ss,omitempty" ms:"s" cbor:"s"`
}

type ProductProperties struct {
	ID         int16                         `ms:"i" cbor:"i"`
	Nombre     string                        `ms:"n" cbor:"n"`
	Options    []ProductProperty           `ms:"o" cbor:"o"`
	Status     int8                          `ms:"s" cbor:"s"`
	OptionsMap map[string]*ProductProperty `json:"-" ms:"-"`
}

type Warehouse struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   int32           `db:"empresa_id,pk"`
	ID          int32           `db:"id,pk"`
	SedeID      int32           `db:"sede_id"`
	Nombre      string          `db:"nombre"`
	Descripcion string          `db:"descripcion"`
	Layout      []WarehouseLayout `db:"layout"`
	// Propiedades generales
	Status    int8   `json:"ss,omitempty" db:"status,view"`
	Updated   int32  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32  `json:",omitempty" db:"updated_by"`
	Created   int32  `json:",omitempty" db:"created"`
	CreatedBy int32  `json:",omitempty" db:"created_by"`
	Ciudad    string `json:",omitempty"`
}

type WarehouseTable struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   db.Col[WarehouseTable, int32]
	ID          db.Col[WarehouseTable, int32]
	SedeID      db.Col[WarehouseTable, int32]
	Nombre      db.Col[WarehouseTable, string]
	Descripcion db.Col[WarehouseTable, string]
	Layout      db.Col[WarehouseTable, []WarehouseLayout]
	Status      db.Col[WarehouseTable, int8]
	Updated     db.Col[WarehouseTable, int32]
	UpdatedBy   db.Col[WarehouseTable, int32]
	Created     db.Col[WarehouseTable, int32]
	CreatedBy   db.Col[WarehouseTable, int32]
}

func (e WarehouseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "almacenes",
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
	ID      int16                 `ms:"i" cbor:"i" json:"id,omitempty"`
	Name    string                `ms:"n" cbor:"n" json:"nm,omitempty"`
	RowCant int8                  `ms:"r" cbor:"r" json:"rc,omitempty"`
	ColCant int8                  `ms:"c" cbor:"c" json:"cc,omitempty"`
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
	Nombre      string `db:"nombre"`
	Descripcion string `db:"descripcion"`
	Direccion   string `db:"direccion"`
	CiudadID    string `db:"pais_ciudad_id"`
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
	Nombre      db.Col[SiteTable, string]
	Descripcion db.Col[SiteTable, string]
	Direccion   db.Col[SiteTable, string]
	CiudadID    db.Col[SiteTable, string] `db:"pais_ciudad_id"`
	Status      db.Col[SiteTable, int8]
	Updated     db.Col[SiteTable, int32]
	UpdatedBy   db.Col[SiteTable, int32]
	Created     db.Col[SiteTable, int32]
	CreatedBy   db.Col[SiteTable, int32]
}

func (e SiteTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sedes",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
