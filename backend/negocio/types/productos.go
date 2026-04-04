package types

import (
	"app/core"
	"app/db"
)

type ProductoImagen struct {
	Name        string `ms:"n" json:"n" cbor:"n"`
	Descripcion string `ms:"d" json:"d" cbor:"d"`
}

type AlmacenStockMin struct {
	WarehouseID int32 `cbor:"a" json:"a"`
	Cantidad  int32 `cbor:"c" json:"c"`
}

type ProductoPesentacion struct {
	ID               int16  `ms:"i" json:"id,omitempty" cbor:"i"`
	AtributoID       int16  `ms:"a" json:"at,omitempty" cbor:"a"`
	Name             string `ms:"n" json:"nm,omitempty" cbor:"n"`
	Color            string `ms:"c" json:"cl,omitempty" cbor:"c"`
	Precio           int32  `ms:"p" json:"pc,omitempty" cbor:"p"`
	DiferenciaPrecio int32  `ms:"d" json:"pd,omitempty" cbor:"d"`
	Status           int8   `ms:"s" json:"ss,omitempty" cbor:"s"`
}

type Producto struct {
	db.TableStruct[ProductoTable, Producto]
	EmpresaID      int32   `json:",omitempty"`
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
	NombreHash     int32   `json:",omitempty"`

	Propiedades    []ProductoPropiedades `json:",omitempty"`
	Presentaciones []ProductoPesentacion `json:",omitempty"`
	Images         []ProductoImagen      `json:",omitempty"`
	Stock          []AlmacenStockMin     `json:",omitempty"`
	StockReservado []AlmacenStockMin     `json:",omitempty"`
	StockStatus    int8                  `json:",omitempty"`
	NameUpdated    int32                 `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	/* concatenada con la Empresa-ID para ser indexadas*/
	CategoriasConStock []int32 `json:",omitempty"`
	CacheVersion       uint8   `json:"ccv,omitempty"`
}

type ProductoTable struct {
	db.TableStruct[ProductoTable, Producto]
	EmpresaID          db.Col[ProductoTable, int32]
	ID                 db.Col[ProductoTable, int32]
	Nombre             db.Col[ProductoTable, string]
	Descripcion        db.Col[ProductoTable, string]
	ContentHTML        db.Col[ProductoTable, string]
	CategoriasIDs      db.ColSlice[ProductoTable, int32] `db:"categorias_ids"`
	MarcaID            db.Col[ProductoTable, int32]
	Params             db.ColSlice[ProductoTable, int8] `db:"params_ids"`
	Precio             db.Col[ProductoTable, int32]
	MonedaID           db.Col[ProductoTable, int16]
	UnidadID           db.Col[ProductoTable, int16]
	Descuento          db.Col[ProductoTable, float32]
	PrecioFinal        db.Col[ProductoTable, int32]
	Peso               db.Col[ProductoTable, float32]
	Volumen            db.Col[ProductoTable, float32]
	SbnCantidad        db.Col[ProductoTable, int32]
	SbnUnidad          db.Col[ProductoTable, string]
	SbnPrecio          db.Col[ProductoTable, int32]
	SbnDescuento       db.Col[ProductoTable, float32]
	SbnPrecioFinal     db.Col[ProductoTable, int32]
	NombreHash         db.Col[ProductoTable, int32]
	Propiedades        db.Col[ProductoTable, []ProductoPropiedades]
	Presentaciones     db.Col[ProductoTable, []ProductoPesentacion]
	Images             db.Col[ProductoTable, []ProductoImagen]
	Stock              db.Col[ProductoTable, []AlmacenStockMin]
	StockReservado     db.Col[ProductoTable, []AlmacenStockMin]
	StockStatus        db.Col[ProductoTable, int8]
	NameUpdated        db.Col[ProductoTable, int32]
	Status             db.Col[ProductoTable, int8]
	Updated            db.Col[ProductoTable, int32]
	UpdatedBy          db.Col[ProductoTable, int32]
	Created            db.Col[ProductoTable, int32]
	CreatedBy          db.Col[ProductoTable, int32]
	CategoriasConStock db.ColSlice[ProductoTable, int32]
}

func (e *Producto) FillCategoriasConStock() {
	e.CategoriasConStock = nil
	if e.StockStatus > 0 {
		for _, cid := range e.CategoriasIDs {
			e.CategoriasConStock = append(e.CategoriasConStock, e.EmpresaID*10000+cid)
		}
	}
}

func (e *Producto) SelfParse() {
	e.NombreHash = core.BasicHashInt(core.NormalizeString(&e.Nombre))
}

func (e ProductoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "productos",
		Partition:        e.EmpresaID,
		UseSequences:     true,
		SaveCacheVersion: true,
		Keys:             []db.Coln{e.ID.Autoincrement(0)},
		GlobalIndexes:    [][]db.Coln{{e.CategoriasConStock}},
		Indexes:          [][]db.Coln{{e.NameUpdated}, {e.NombreHash}},
		Views: []db.View{
			{Keys: []db.Coln{e.Status}, KeepPart: true},
			{Keys: []db.Coln{e.StockStatus}, KeepPart: true},
			{Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ProductoPropiedad struct {
	ID     int16  `json:"id,omitempty" ms:"i" cbor:"i"`
	Nombre string `json:"nm,omitempty" ms:"n" cbor:"n"`
	Status int8   `json:"ss,omitempty" ms:"s" cbor:"s"`
}

type ProductoPropiedades struct {
	ID         int16                         `ms:"i" cbor:"i"`
	Nombre     string                        `ms:"n" cbor:"n"`
	Options    []ProductoPropiedad           `ms:"o" cbor:"o"`
	Status     int8                          `ms:"s" cbor:"s"`
	OptionsMap map[string]*ProductoPropiedad `json:"-" ms:"-"`
}

type Almacen struct {
	db.TableStruct[AlmacenTable, Almacen]
	EmpresaID   int32           `db:"empresa_id,pk"`
	ID          int32           `db:"id,pk"`
	SedeID      int32           `db:"sede_id"`
	Nombre      string          `db:"nombre"`
	Descripcion string          `db:"descripcion"`
	Layout      []AlmacenLayout `db:"layout"`
	// Propiedades generales
	Status    int8   `json:"ss,omitempty" db:"status,view"`
	Updated   int32  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32  `json:",omitempty" db:"updated_by"`
	Created   int32  `json:",omitempty" db:"created"`
	CreatedBy int32  `json:",omitempty" db:"created_by"`
	Ciudad    string `json:",omitempty"`
}

type AlmacenTable struct {
	db.TableStruct[AlmacenTable, Almacen]
	EmpresaID   db.Col[AlmacenTable, int32]
	ID          db.Col[AlmacenTable, int32]
	SedeID      db.Col[AlmacenTable, int32]
	Nombre      db.Col[AlmacenTable, string]
	Descripcion db.Col[AlmacenTable, string]
	Layout      db.Col[AlmacenTable, []AlmacenLayout]
	Status      db.Col[AlmacenTable, int8]
	Updated     db.Col[AlmacenTable, int32]
	UpdatedBy   db.Col[AlmacenTable, int32]
	Created     db.Col[AlmacenTable, int32]
	CreatedBy   db.Col[AlmacenTable, int32]
}

func (e AlmacenTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "almacenes",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Views: []db.View{
			{Keys: []db.Coln{e.Status}, KeepPart: true},
			{Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type AlmacenLayout struct {
	ID      int16                 `ms:"i" cbor:"i" json:"id,omitempty"`
	Name    string                `ms:"n" cbor:"n" json:"nm,omitempty"`
	RowCant int8                  `ms:"r" cbor:"r" json:"rc,omitempty"`
	ColCant int8                  `ms:"c" cbor:"c" json:"cc,omitempty"`
	Bloques []AlmacenLayoutBloque `ms:"b" cbor:"b" json:"bl,omitempty"`
}

type AlmacenLayoutBloque struct {
	Row    int8   `json:"rw" ms:"r" cbor:"r"`
	Column int8   `json:"co" ms:"c" cbor:"c"`
	Name   string `json:"nm" ms:"n" cbor:"n"`
}

type Sede struct {
	db.TableStruct[SedeTable, Sede]
	EmpresaID   int32  `db:"empresa_id,pk"`
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

type SedeTable struct {
	db.TableStruct[SedeTable, Sede]
	EmpresaID   db.Col[SedeTable, int32]
	ID          db.Col[SedeTable, int32]
	Nombre      db.Col[SedeTable, string]
	Descripcion db.Col[SedeTable, string]
	Direccion   db.Col[SedeTable, string]
	CiudadID    db.Col[SedeTable, string] `db:"pais_ciudad_id"`
	Status      db.Col[SedeTable, int8]
	Updated     db.Col[SedeTable, int32]
	UpdatedBy   db.Col[SedeTable, int32]
	Created     db.Col[SedeTable, int32]
	CreatedBy   db.Col[SedeTable, int32]
}

func (e SedeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sedes",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Views: []db.View{
			{Keys: []db.Coln{e.Status}, KeepPart: true},
			{Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
