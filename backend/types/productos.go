package types

import "app/db2"

type ProductoImagen struct {
	Name        string `ms:"n" json:"n" cbor:"n"`
	Descripcion string `ms:"d" json:"d" cbor:"d"`
}

type AlmacenStockMin struct {
	AlmacenID int32 `cbor:"a" json:"a"`
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
	db2.TableStruct[ProductoTable, Producto]
	EmpresaID      int32   `json:",omitempty" db:"empresa_id,pk"`
	ID             int32   `db:"id,pk"`
	TempID         int32   `json:"-" db:"-"`
	Nombre         string  `db:"nombre"`
	Descripcion    string  `json:",omitempty" db:"descripcion"`
	ContentHTML    string  `json:",omitempty" db:"content_html"`
	CategoriasIDs  []int32 `json:",omitempty" db:"categorias_ids"`
	MarcaID        int32   `json:",omitempty" db:"marca_id"`
	Params         []int8  `json:",omitempty" db:"params_ids"`
	Precio         int32   `json:",omitempty" db:"precio"`
	MonedaID       int16   `json:",omitempty" db:"moneda_id"`
	UnidadID       int16   `json:",omitempty" db:"unidad_id"`
	Descuento      float32 `json:",omitempty" db:"descuento"`
	PrecioFinal    int32   `json:",omitempty" db:"precio_final"`
	Peso           float32 `json:",omitempty" db:"peso"`
	Volumen        float32 `json:",omitempty" db:"volumen"`
	SbnCantidad    int32   `json:",omitempty" db:"sbn_cantidad"`
	SbnUnidad      string  `json:",omitempty" db:"sbn_unidad"`
	SbnPrecio      int32   `json:",omitempty" db:"sbn_precio"`
	SbnDescuento   float32 `json:",omitempty" db:"sbn_decuento"`
	SbnPrecioFinal int32   `json:",omitempty" db:"sbn_precio_final"`

	Propiedades    []ProductoPropiedades `json:",omitempty" db:"propiedades"`
	Presentaciones []ProductoPesentacion `json:",omitempty" db:"presentaciones"`
	Images         []ProductoImagen      `json:",omitempty" db:"images"`
	Stock          []AlmacenStockMin     `json:",omitempty" db:"stock"`
	StockReservado []AlmacenStockMin     `json:",omitempty" db:"stock_reservado"`
	StockStatus    int8                  `json:",omitempty" db:"stock_status,view.1"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int64 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
	/* concatenada con la Empresa-ID para ser indexadas*/
	CategoriasConStock []int32 `json:",omitempty" db:"categorias_con_stock,gindex"`
}

type ProductoTable struct {
	db2.TableStruct[ProductoTable, Producto]
	EmpresaID          db2.Col[ProductoTable, int32]
	ID                 db2.Col[ProductoTable, int32]
	Nombre             db2.Col[ProductoTable, string]
	Descripcion        db2.Col[ProductoTable, string]
	ContentHTML        db2.Col[ProductoTable, string]
	CategoriasIDs      db2.ColSlice[ProductoTable, int32] `db:"categorias_ids"`
	MarcaID            db2.Col[ProductoTable, int32]
	Params             db2.ColSlice[ProductoTable, int8] `db:"params_ids"`
	Precio             db2.Col[ProductoTable, int32]
	MonedaID           db2.Col[ProductoTable, int16]
	UnidadID           db2.Col[ProductoTable, int16]
	Descuento          db2.Col[ProductoTable, float32]
	PrecioFinal        db2.Col[ProductoTable, int32]
	Peso               db2.Col[ProductoTable, float32]
	Volumen            db2.Col[ProductoTable, float32]
	SbnCantidad        db2.Col[ProductoTable, int32]
	SbnUnidad          db2.Col[ProductoTable, string]
	SbnPrecio          db2.Col[ProductoTable, int32]
	SbnDescuento       db2.Col[ProductoTable, float32]
	SbnPrecioFinal     db2.Col[ProductoTable, int32]
	Propiedades        db2.Col[ProductoTable, []ProductoPropiedades]
	Presentaciones     db2.Col[ProductoTable, []ProductoPesentacion]
	Images             db2.Col[ProductoTable, []ProductoImagen]
	Stock              db2.Col[ProductoTable, []AlmacenStockMin]
	StockReservado     db2.Col[ProductoTable, []AlmacenStockMin]
	StockStatus        db2.Col[ProductoTable, int8]
	Status             db2.Col[ProductoTable, int8]
	Updated            db2.Col[ProductoTable, int64]
	UpdatedBy          db2.Col[ProductoTable, int32]
	Created            db2.Col[ProductoTable, int64]
	CreatedBy          db2.Col[ProductoTable, int32]
	CategoriasConStock db2.ColSlice[ProductoTable, int32]
}

func (e *Producto) FillCategoriasConStock() {
	e.CategoriasConStock = nil
	if e.StockStatus > 0 {
		for _, cid := range e.CategoriasIDs {
			e.CategoriasConStock = append(e.CategoriasConStock, e.EmpresaID*10000+cid)
		}
	}
}

func (e ProductoTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:          "productos",
		Partition:     e.EmpresaID,
		Keys:          []db2.Coln{e.ID},
		GlobalIndexes: []db2.Coln{e.CategoriasConStock},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Status}, KeepPart: true},
			{Cols: []db2.Coln{e.StockStatus}, KeepPart: true},
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
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
	db2.TableStruct[AlmacenTable, Almacen]
	EmpresaID   int32           `db:"empresa_id,pk"`
	ID          int32           `db:"id,pk"`
	SedeID      int32           `db:"sede_id"`
	Nombre      string          `db:"nombre"`
	Descripcion string          `db:"descripcion"`
	Layout      []AlmacenLayout `db:"layout"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int64 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
}

type AlmacenTable struct {
	db2.TableStruct[AlmacenTable, Almacen]
	EmpresaID   db2.Col[AlmacenTable, int32]
	ID          db2.Col[AlmacenTable, int32]
	SedeID      db2.Col[AlmacenTable, int32]
	Nombre      db2.Col[AlmacenTable, string]
	Descripcion db2.Col[AlmacenTable, string]
	Layout      db2.Col[AlmacenTable, []AlmacenLayout]
	Status      db2.Col[AlmacenTable, int8]
	Updated     db2.Col[AlmacenTable, int64]
	UpdatedBy   db2.Col[AlmacenTable, int32]
	Created     db2.Col[AlmacenTable, int64]
	CreatedBy   db2.Col[AlmacenTable, int32]
}

func (e AlmacenTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:         "almacenes",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Status}, KeepPart: true},
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
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
	db2.TableStruct[SedeTable, Sede]
	EmpresaID   int32  `db:"empresa_id,pk"`
	ID          int32  `db:"id,pk"`
	Nombre      string `db:"nombre"`
	Descripcion string `db:"descripcion"`
	Direccion   string `db:"direccion"`
	CiudadID    string `db:"pais_ciudad_id"`
	Status      int8   `json:"ss,omitempty" db:"status,view"`
	Updated     int64  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy   int32  `json:",omitempty" db:"updated_by"`
	Created     int64  `json:",omitempty" db:"created"`
	CreatedBy   int32  `json:",omitempty" db:"created_by"`
	Ciudad      string `json:",omitempty"`
}

type SedeTable struct {
	db2.TableStruct[SedeTable, Sede]
	EmpresaID   db2.Col[SedeTable, int32]
	ID          db2.Col[SedeTable, int32]
	Nombre      db2.Col[SedeTable, string]
	Descripcion db2.Col[SedeTable, string]
	Direccion   db2.Col[SedeTable, string]
	CiudadID    db2.Col[SedeTable, string] `db:"pais_ciudad_id"`
	Status      db2.Col[SedeTable, int8]
	Updated     db2.Col[SedeTable, int64]
	UpdatedBy   db2.Col[SedeTable, int32]
	Created     db2.Col[SedeTable, int64]
	CreatedBy   db2.Col[SedeTable, int32]
}

func (e SedeTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "sedes",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Status}, KeepPart: true},
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type AlmacenProducto struct {
	db2.TableStruct[AlmacenProductoTable, AlmacenProducto]
	EmpresaID int32 `json:",omitempty" db:"empresa_id,pk"`
	// [Almacen-ID] [status] [Producto-ID] [SKU] [Lote]
	ID             string  `db:"id,pk"`
	SKU            string  `json:",omitempty" db:"sku,index"`
	Lote           string  `json:",omitempty" db:"lote,index"`
	AlmacenID      int32   `json:",omitempty" db:"almacen_id,view.1"`
	ProductoID     int32   `json:",omitempty" db:"producto_id,view"`
	PresentacionID int16   `json:",omitempty" db:"presentacion_id"`
	Cantidad       int32   `json:",omitempty" db:"cantidad"`
	SubCantidad    int32   `json:",omitempty" db:"sub_cantidad"`
	CostoUn        float32 `json:",omitempty" db:"costo_un"`
	Updated        int32   `json:"upd,omitempty" db:"updated,view.1"`
	UpdatedBy      int32   `json:",omitempty" db:"updated_by"`
	Status         int8    `json:"ss,omitempty" db:"status,view,view.1"`
}

type AlmacenProductoTable struct {
	db2.TableStruct[AlmacenProductoTable, AlmacenProducto]
	EmpresaID      db2.Col[AlmacenProductoTable, int32]
	ID             db2.Col[AlmacenProductoTable, string]
	SKU            db2.Col[AlmacenProductoTable, string]
	Lote           db2.Col[AlmacenProductoTable, string]
	AlmacenID      db2.Col[AlmacenProductoTable, int32]
	ProductoID     db2.Col[AlmacenProductoTable, int32]
	PresentacionID db2.Col[AlmacenProductoTable, int16]
	Cantidad       db2.Col[AlmacenProductoTable, int32]
	SubCantidad    db2.Col[AlmacenProductoTable, int32]
	CostoUn        db2.Col[AlmacenProductoTable, float32]
	Updated        db2.Col[AlmacenProductoTable, int32]
	UpdatedBy      db2.Col[AlmacenProductoTable, int32]
	Status         db2.Col[AlmacenProductoTable, int8]
}

func (e AlmacenProductoTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:         "almacen_producto",
		Partition:    e.EmpresaID,
		Keys:         []db2.Coln{e.ID},
		LocalIndexes: []db2.Coln{e.SKU, e.Lote},
		Views: []db2.View{
			{
				Cols:      []db2.Coln{e.ProductoID, e.Status},
				KeepPart:  true,
				ConcatI32: []int8{1},
			},
			{
				Cols:      []db2.Coln{e.AlmacenID, e.Status, e.Updated},
				KeepPart:  true,
				ConcatI64: []int8{1, 9},
			},
		},
	}
}

func (e *AlmacenProducto) SelfParse() {
	e.ID = Concat62(e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote)
}

/*
func (e *AlmacenProducto) GetView(view int8) any {
	if view == 1 {
		return int64(e.AlmacenID)*1e9 + int64(e.Updated)
	} else {
		return 0
	}
}
*/

type AlmacenMovimiento struct {
	db2.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID int32 `json:",omitempty" db:"empresa_id,pk"`
	// [Almacen-ID] + [Created] + [Ramdom Number]
	ID                 int64  `db:"id,pk"`
	SKU                string `json:",omitempty" db:"sku,view"`
	Lote               string `json:",omitempty" db:"lote,view.1"`
	AlmacenID          int32  `json:",omitempty" db:"almacen_id"`
	AlmacenRefID       int32  `json:",omitempty" db:"almacen_ref_id,view.2"`
	AlmacenRefCantidad int32  `json:",omitempty" db:"almacen_ref_cantidad"`
	VentaID            int32  `json:",omitempty" db:"venta_id"`
	ProductoID         int32  `json:",omitempty" db:"producto_id"`
	PresentacionID     int16  `json:",omitempty" db:"presentacion_id"`
	Cantidad           int32  `json:",omitempty" db:"cantidad"`
	AlmacenCantidad    int32  `json:",omitempty" db:"almacen_cantidad"`
	SubCantidad        int32  `json:",omitempty" db:"sub_cantidad"`
	Tipo               int8   `json:",omitempty" db:"tipo"`
	Created            int32  `json:",omitempty" db:"created,view.2"`
	CreatedBy          int32  `json:",omitempty" db:"created_by"`
}

type AlmacenMovimientoTable struct {
	db2.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID          db2.Col[AlmacenMovimientoTable, int32]
	ID                 db2.Col[AlmacenMovimientoTable, int64]
	SKU                db2.Col[AlmacenMovimientoTable, string]
	Lote               db2.Col[AlmacenMovimientoTable, string]
	AlmacenID          db2.Col[AlmacenMovimientoTable, int32]
	AlmacenRefID       db2.Col[AlmacenMovimientoTable, int32]
	AlmacenRefCantidad db2.Col[AlmacenMovimientoTable, int32]
	VentaID            db2.Col[AlmacenMovimientoTable, int32]
	ProductoID         db2.Col[AlmacenMovimientoTable, int32]
	PresentacionID     db2.Col[AlmacenMovimientoTable, int16]
	Cantidad           db2.Col[AlmacenMovimientoTable, int32]
	AlmacenCantidad    db2.Col[AlmacenMovimientoTable, int32]
	SubCantidad        db2.Col[AlmacenMovimientoTable, int32]
	Tipo               db2.Col[AlmacenMovimientoTable, int8]
	Created            db2.Col[AlmacenMovimientoTable, int32]
	CreatedBy          db2.Col[AlmacenMovimientoTable, int32]
}

func (e AlmacenMovimientoTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "almacen_movimiento",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.SKU}, KeepPart: true},
			{Cols: []db2.Coln{e.Lote}, KeepPart: true},
			{Cols: []db2.Coln{e.AlmacenRefID, e.Created}, ConcatI64: []int8{9}},
		},
	}
}

type MovimientoInterno struct {
	ProductoID           int32
	PresentacionID       int16
	SKU                  string
	Lote                 string
	AlmacenID            int32
	AlmacenDestinoID     int32
	ReemplazarCantidad   bool
	Cantidad             int32
	SubCantidad          int32
	ModificarCantidad    int32
	ModificarSubCantidad int32
}

func (e *MovimientoInterno) GetAlmacenProductoID() string {
	return Concat62(e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote)
}
