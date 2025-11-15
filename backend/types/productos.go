package types

import "app/db"

type ProductoImagen struct {
	Name        string `ms:"n" json:"n"`
	Descripcion string `ms:"d" json:"d"`
}

type AlmacenStockMin struct {
	AlmacenID int32 `cbor:"1,keyasint" json:"a"`
	Cantidad  int32 `cbor:"2,keyasint" json:"c"`
}

type Producto struct {
	TAGS          `table:"productos"`
	EmpresaID     int32 `json:",omitempty"`
	ID            int32
	TempID        int32 `json:"-"`
	Nombre        string
	Descripcion   string  `json:",omitempty"`
	ContentHTML   string  `json:",omitempty"`
	CategoriasIDs []int32 `json:",omitempty"`
	MarcaID       int32   `json:",omitempty"`
	Params        []int8  `json:",omitempty"`
	Precio        int32   `json:",omitempty"`
	MonedaID      int16   `json:",omitempty"`
	UnidadID      int16   `json:",omitempty"`
	Descuento     float32 `json:",omitempty"`
	PrecioFinal   int32   `json:",omitempty"`
	Peso          float32 `json:",omitempty"`
	Volumen       float32 `json:",omitempty"`
	SbnCantidad   int32   `json:",omitempty"`
	SbnUnidad     string  `json:",omitempty"`
	SbnPrecio     int32   `json:",omitempty"`
	SbnDescuento  float32 `json:",omitempty"`
	SbnPreciFinal int32   `json:",omitempty"`

	Propiedades    []ProductoPropiedades `json:",omitempty"`
	Images         []ProductoImagen      `json:",omitempty"`
	Stock          []AlmacenStockMin     `json:",omitempty"`
	StockReservado []AlmacenStockMin     `json:",omitempty"`
	StockStatus    int8                  `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int64 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int64 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	/* concatenada con la Empresa-ID para ser indexadas*/
	CategoriasConStock []int32 `json:",omitempty"`
}

type _e = Producto

func (e _e) EmpresaID_() db.CoI32          { return db.CoI32{"empresa_id"} }
func (e _e) ID_() db.CoI32                 { return db.CoI32{"id"} }
func (e _e) TempID_() db.CoI32             { return db.CoI32{"temp_id"} }
func (e _e) Nombre_() db.CoStr             { return db.CoStr{"nombre"} }
func (e _e) Descripcion_() db.CoStr        { return db.CoStr{"descripcion"} }
func (e _e) ContentHTML_() db.CoStr        { return db.CoStr{"content_html"} }
func (e _e) CategoriasIDs_() db.CsI32      { return db.CsI32{"categorias_ids"} }
func (e _e) CategoriasConStock_() db.CsI32 { return db.CsI32{"categorias_con_stock"} }
func (e _e) Params_() db.CsI8              { return db.CsI8{"params_ids"} }
func (e _e) Precio_() db.CoI32             { return db.CoI32{"precio"} }
func (e _e) Descuento_() db.CoF32          { return db.CoF32{"descuento"} }
func (e _e) PrecioFinal_() db.CoI32        { return db.CoI32{"precio_final"} }
func (e _e) Peso_() db.CoF32               { return db.CoF32{"peso"} }
func (e _e) Volumen_() db.CoF32            { return db.CoF32{"volumen"} }
func (e _e) MonedaID_() db.CoI16           { return db.CoI16{"moneda_id"} }
func (e _e) UnidadID_() db.CoI16           { return db.CoI16{"unidad_id"} }
func (e _e) SbnCantidad_() db.CoI32        { return db.CoI32{"sbn_cantidad"} }
func (e _e) SbnUnidad_() db.CoStr          { return db.CoStr{"sbn_unidad"} }
func (e _e) MarcaID_() db.CoI32            { return db.CoI32{"marca_id"} }
func (e _e) SbnPrecio_() db.CoI32          { return db.CoI32{"sbn_precio"} }
func (e _e) SbnDescuento_() db.CoF32       { return db.CoF32{"sbn_decuento"} }
func (e _e) SbnPreciFinal_() db.CoI32      { return db.CoI32{"sbn_precio_final"} }
func (e _e) Propiedades_() db.CoAny        { return db.CoAny{"propiedades"} }
func (e _e) Images_() db.CoAny             { return db.CoAny{"images"} }
func (e _e) Status_() db.CoI8              { return db.CoI8{"status"} }
func (e _e) Updated_() db.CoI64            { return db.CoI64{"updated"} }
func (e _e) UpdatedBy_() db.CoI32          { return db.CoI32{"updated_by"} }
func (e _e) Created_() db.CoI64            { return db.CoI64{"created"} }
func (e _e) CreatedBy_() db.CoI32          { return db.CoI32{"created_by"} }
func (e _e) Stock_() db.CoAny              { return db.CoAny{"stock"} }
func (e _e) StockReservado_() db.CoAny     { return db.CoAny{"stock_reservado"} }
func (e _e) StockStatus_() db.CoI8         { return db.CoI8{"stock_status"} }

func (e *Producto) FillCategoriasConStock() {
	e.CategoriasConStock = nil
	if e.StockStatus > 0 {
		for _, cid := range e.CategoriasIDs {
			e.CategoriasConStock = append(e.CategoriasConStock, e.EmpresaID*10000+cid)
		}
	}
}

func (e Producto) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:          "productos",
		Partition:     e.EmpresaID_(),
		Keys:          []db.Coln{e.ID_()},
		GlobalIndexes: []db.Coln{e.CategoriasConStock_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.StockStatus_()}, KeepPart: true},
			{Cols: []db.Coln{e.Updated_()}, KeepPart: true},
		},
	}
}

type ProductoPropiedad struct {
	ID     int16  `json:"id,omitempty" ms:"i"`
	Nombre string `json:"nm,omitempty" ms:"n"`
	Status int8   `json:"ss,omitempty" ms:"s"`
}

type ProductoPropiedades struct {
	ID         int16                         `ms:"i"`
	Nombre     string                        `ms:"n"`
	Options    []ProductoPropiedad           `ms:"o"`
	Status     int8                          `ms:"s"`
	OptionsMap map[string]*ProductoPropiedad `json:"-" ms:"-"`
}

type Almacen struct {
	TAGS        `table:"almacenes"`
	EmpresaID   int32
	ID          int32
	SedeID      int32
	Nombre      string
	Descripcion string
	Layout      []AlmacenLayout
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int64 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int64 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
}

type _c = Almacen

func (e _c) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e _c) ID_() db.CoI32          { return db.CoI32{"id"} }
func (e _c) SedeID_() db.CoI32      { return db.CoI32{"sede_id"} }
func (e _c) Nombre_() db.CoStr      { return db.CoStr{"nombre"} }
func (e _c) Descripcion_() db.CoStr { return db.CoStr{"descripcion"} }
func (e _c) Layout_() db.CoAny      { return db.CoAny{"layout"} }
func (e _c) Status_() db.CoI8       { return db.CoI8{"status"} }
func (e _c) Updated_() db.CoI64     { return db.CoI64{"updated"} }
func (e _c) UpdatedBy_() db.CoI32   { return db.CoI32{"updated_by"} }
func (e _c) Created_() db.CoI64     { return db.CoI64{"created"} }
func (e _c) CreatedBy_() db.CoI32   { return db.CoI32{"created_by"} }

func (e Almacen) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "almacenes",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.Updated_()}, KeepPart: true},
		},
	}
}

type AlmacenLayout struct {
	ID      int16                 `ms:"i" cbor:"1,keyasint,omitempty"`
	Name    string                `ms:"n" cbor:"2,keyasint,omitempty"`
	RowCant int8                  `ms:"r" cbor:"3,keyasint,omitempty"`
	ColCant int8                  `ms:"c" cbor:"4,keyasint,omitempty"`
	Bloques []AlmacenLayoutBloque `ms:"b" cbor:"5,keyasint,omitempty"`
}

type AlmacenLayoutBloque struct {
	Row    int8   `json:"rw" ms:"r" cbor:"1,keyasint,omitempty"`
	Column int8   `json:"co" ms:"c" cbor:"2,keyasint,omitempty"`
	Name   string `json:"nm" ms:"n" cbor:"2,keyasint,omitempty"`
}

type Sede struct {
	TAGS        `table:"sedes"`
	EmpresaID   int32  `db:"empresa_id,pk"`
	ID          int32  `db:"id,pk"`
	Nombre      string `db:"nombre"`
	Descripcion string `db:"descripcion"`
	Direccion   string `db:"direccion"`
	CiudadID    string `db:"pais_ciudad_id"`
	Status      int8   `json:"ss" db:"status,view"`
	Updated     int64  `json:"upd" db:"updated,view"`
	UpdatedBy   int32  `db:"updated_by"`
	Created     int64  `db:"created"`
	CreatedBy   int32  `db:"created_by"`
	Ciudad      string `json:"omitempty"`
}

type _d = Sede

func (e _d) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e _d) ID_() db.CoI32          { return db.CoI32{"id"} }
func (e _d) Nombre_() db.CoStr      { return db.CoStr{"nombre"} }
func (e _d) Descripcion_() db.CoStr { return db.CoStr{"descripcion"} }
func (e _d) Direccion_() db.CoStr   { return db.CoStr{"direccion"} }
func (e _d) CiudadID_() db.CoStr    { return db.CoStr{"pais_ciudad_id"} }
func (e _d) Status_() db.CoI8       { return db.CoI8{"status"} }
func (e _d) Updated_() db.CoI64     { return db.CoI64{"updated"} }
func (e _d) UpdatedBy_() db.CoI32   { return db.CoI32{"updated_by"} }
func (e _d) Created_() db.CoI64     { return db.CoI64{"created"} }
func (e _d) CreatedBy_() db.CoI32   { return db.CoI32{"created_by"} }

func (e Sede) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "sedes",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.Updated_()}, KeepPart: true},
		},
	}
}

type AlmacenProducto struct {
	TAGS      `table:"almacen_producto"`
	EmpresaID int32 `json:",omitempty"`
	// [Almacen-ID] [status] [Producto-ID] [SKU] [Lote]
	ID          string  `db:"id,pk"`
	SKU         string  `json:",omitempty"`
	Lote        string  `json:",omitempty"`
	AlmacenID   int32   `json:",omitempty"`
	ProductoID  int32   `json:",omitempty"`
	Cantidad    int32   `json:",omitempty"`
	SubCantidad int32   `json:",omitempty"`
	CostoUn     float32 `json:",omitempty"`
	Updated     int32   `json:"upd,omitempty"`
	UpdatedBy   int32   `json:",omitempty"`
	Status      int8    `json:"ss,omitempty"`
}

type alp = AlmacenProducto

func (e alp) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e alp) ID_() db.CoStr          { return db.CoStr{"id"} }
func (e alp) SKU_() db.CoStr         { return db.CoStr{"sku"} }
func (e alp) Lote_() db.CoStr        { return db.CoStr{"lote"} }
func (e alp) AlmacenID_() db.CoI32   { return db.CoI32{"almacen_id"} }
func (e alp) ProductoID_() db.CoI32  { return db.CoI32{"producto_id"} }
func (e alp) Cantidad_() db.CoI32    { return db.CoI32{"cantidad"} }
func (e alp) SubCantidad_() db.CoI32 { return db.CoI32{"sub_cantidad"} }
func (e alp) CostoUn_() db.CoI32     { return db.CoI32{"costo_un"} }
func (e alp) Updated_() db.CoI32     { return db.CoI32{"updated"} }
func (e alp) UpdatedBy_() db.CoI32   { return db.CoI32{"updated_by"} }
func (e alp) Status_() db.CoI8       { return db.CoI8{"status"} }

func (e AlmacenProducto) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "almacen_producto",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.SKU_()}, KeepPart: true},
			{Cols: []db.Coln{e.Lote_()}, KeepPart: true},
			// {Cols: []db.Coln{e.ProductoID_()}, KeepPart: true},
			{Cols: []db.Coln{e.ProductoID_(), e.Status_()}, KeepPart: true, ConcatI32: []int8{1}},
			{Cols: []db.Coln{e.AlmacenID_(), e.Updated_()}, KeepPart: true, ConcatI64: []int8{9}},
			{Cols: []db.Coln{e.AlmacenID_(), e.Status_()}, KeepPart: true, ConcatI32: []int8{1}},
		},
	}
}

func (e *AlmacenProducto) SelfParse() {
	e.ID = Concat62(e.AlmacenID, e.ProductoID, e.SKU, e.Lote)
}

func (e *AlmacenProducto) GetView(view int8) any {
	if view == 1 {
		return int64(e.AlmacenID)*1e9 + int64(e.Updated)
	} else {
		return 0
	}
}

type AlmacenMovimiento struct {
	TAGS      `table:"almacen_movimiento"`
	EmpresaID int32 `json:",omitempty" db:"empresa_id,pk"`
	// [Almacen-ID] + [Created] + [Ramdom Number]
	ID                 int64  `db:"id,pk"`
	SKU                string `json:",omitempty" db:"sku,view"`
	Lote               string `json:",omitempty" db:"lote,view"`
	AlmacenID          int32  `json:",omitempty" db:"almacen_id"`
	AlmacenRefID       int32  `json:",omitempty" db:"almacen_ref_id,view.1"`
	AlmacenRefCantidad int32  `json:",omitempty" db:"almacen_ref_cantidad"`
	VentaID            int32  `json:",omitempty" db:"venta_id"`
	ProductoID         int32  `json:",omitempty" db:"producto_id"`
	Cantidad           int32  `json:",omitempty" db:"cantidad"`
	AlmacenCantidad    int32  `json:",omitempty" db:"almacen_cantidad"`
	SubCantidad        int32  `json:",omitempty" db:"sub_cantidad"`
	Tipo               int8   `json:",omitempty" db:"tipo"`
	Created            int32  `json:",omitempty" db:"created,view.1"`
	CreatedBy          int32  `json:",omitempty" db:"created_by"`
}

func (e *AlmacenMovimiento) GetView(view int8) any {
	if view == 1 {
		return int64(e.AlmacenRefID)*1e9 + int64(e.Created)
	} else {
		return 0
	}
}

type alm = AlmacenMovimiento

func (e alm) EmpresaID_() db.CoI32       { return db.CoI32{"empresa_id"} }
func (e alm) ID_() db.CoI64              { return db.CoI64{"id"} }
func (e alm) SKU_() db.CoStr             { return db.CoStr{"sku"} }
func (e alm) Lote_() db.CoStr            { return db.CoStr{"lote"} }
func (e alm) AlmacenID_() db.CoI32       { return db.CoI32{"almacen_id"} }
func (e alm) AlmacenRefID_() db.CoI32    { return db.CoI32{"almacen_ref_id"} }
func (e alm) VentaID_() db.CoI32         { return db.CoI32{"venta_id"} }
func (e alm) ProductoID_() db.CoI32      { return db.CoI32{"producto_id"} }
func (e alm) Cantidad_() db.CoI32        { return db.CoI32{"cantidad"} }
func (e alm) AlmacenCantidad_() db.CoI32 { return db.CoI32{"almacen_cantidad"} }
func (e alm) SubCantidad_() db.CoI32     { return db.CoI32{"sub_cantidad"} }
func (e alm) Tipo_() db.CoI8             { return db.CoI8{"tipo"} }
func (e alm) Created_() db.CoI32         { return db.CoI32{"created"} }
func (e alm) CreatedBy_() db.CoI32       { return db.CoI32{"created_by"} }

func (e alm) AlmacenRefCantidad_() db.CoI32 { return db.CoI32{"almacen_ref_cantidad"} }

func (e alm) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "almacen_movimiento",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.SKU_()}, KeepPart: true},
			{Cols: []db.Coln{e.Lote_()}, KeepPart: true},
			{Cols: []db.Coln{e.AlmacenRefID_(), e.Created_()}, ConcatI64: []int8{9}},
		},
	}
}

type MovimientoInterno struct {
	ProductoID           int32
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
	return Concat62(e.AlmacenID, e.ProductoID, e.SKU, e.Lote)
}
