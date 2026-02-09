package types

import "app/db"

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
	db.TableStruct[ProductoTable, Producto]
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
	Propiedades        db.Col[ProductoTable, []ProductoPropiedades]
	Presentaciones     db.Col[ProductoTable, []ProductoPesentacion]
	Images             db.Col[ProductoTable, []ProductoImagen]
	Stock              db.Col[ProductoTable, []AlmacenStockMin]
	StockReservado     db.Col[ProductoTable, []AlmacenStockMin]
	StockStatus        db.Col[ProductoTable, int8]
	Status             db.Col[ProductoTable, int8]
	Updated            db.Col[ProductoTable, int64]
	UpdatedBy          db.Col[ProductoTable, int32]
	Created            db.Col[ProductoTable, int64]
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

func (e ProductoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:          "productos",
		Partition:     e.EmpresaID,
		UseSequences:  true,
		Keys:          []db.Coln{e.ID.Autoincrement(0)},
		GlobalIndexesDeprecated: []db.Coln{e.CategoriasConStock},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.StockStatus}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
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
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view"`
		UpdatedBy   int32  `json:",omitempty" db:"updated_by"`
		Created     int64  `json:",omitempty" db:"created"`
		CreatedBy   int32  `json:",omitempty" db:"created_by"`
		Ciudad      string `json:",omitempty"`
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
	Updated     db.Col[AlmacenTable, int64]
	UpdatedBy   db.Col[AlmacenTable, int32]
	Created     db.Col[AlmacenTable, int64]
	CreatedBy   db.Col[AlmacenTable, int32]
}

func (e AlmacenTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "almacenes",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
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
	Updated     int64  `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy   int32  `json:",omitempty" db:"updated_by"`
	Created     int64  `json:",omitempty" db:"created"`
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
	Updated     db.Col[SedeTable, int64]
	UpdatedBy   db.Col[SedeTable, int32]
	Created     db.Col[SedeTable, int64]
	CreatedBy   db.Col[SedeTable, int32]
}

func (e SedeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "sedes",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type AlmacenProducto struct {
	db.TableStruct[AlmacenProductoTable, AlmacenProducto]
	EmpresaID int32 `json:",omitempty"`
	ID             string
	SKU            string  `json:",omitempty"`
	Lote           string  `json:",omitempty"`
	AlmacenID      int32   `json:",omitempty"`
	ProductoID     int32   `json:",omitempty"`
	PresentacionID int16   `json:",omitempty"`
	Cantidad       int32   `json:",omitempty"`
	SubCantidad    int32   `json:",omitempty"`
	CostoUn        float32 `json:",omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Status         int8    `json:"ss,omitempty"`
}

type AlmacenProductoTable struct {
	db.TableStruct[AlmacenProductoTable, AlmacenProducto]
	EmpresaID      db.Col[AlmacenProductoTable, int32]
	ID             db.Col[AlmacenProductoTable, string]
	SKU            db.Col[AlmacenProductoTable, string]
	Lote           db.Col[AlmacenProductoTable, string]
	AlmacenID      db.Col[AlmacenProductoTable, int32]
	ProductoID     db.Col[AlmacenProductoTable, int32]
	PresentacionID db.Col[AlmacenProductoTable, int16]
	Cantidad       db.Col[AlmacenProductoTable, int32]
	SubCantidad    db.Col[AlmacenProductoTable, int32]
	CostoUn        db.Col[AlmacenProductoTable, float32]
	Updated        db.Col[AlmacenProductoTable, int32]
	UpdatedBy      db.Col[AlmacenProductoTable, int32]
	Status         db.Col[AlmacenProductoTable, int8]
}

func (e AlmacenProductoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "almacen_producto",
		Partition:       e.EmpresaID,
		Keys:            []db.Coln{e.ID},
		KeyConcatenated: []db.Coln{e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote},
		LocalIndexes:    []db.Coln{e.SKU, e.Lote},
		ViewsDeprecated: []db.View{
			{
				Cols:      []db.Coln{e.ProductoID, e.Status},
				KeepPart:  true,
				ConcatI32: []int8{1},
			},
			{
				Cols:      []db.Coln{e.AlmacenID, e.Status, e.Updated},
				KeepPart:  true,
				ConcatI64: []int8{1, 9},
			},
		},
	}
}


type AlmacenMovimiento struct {
	db.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID int32 `json:",omitempty"`
	// [Almacen-ID] + [Created] + [Ramdom Number]
	ID                 int64 
	SKU                string `json:",omitempty"`
	Lote               string `json:",omitempty"`
	AlmacenID          int32  `json:",omitempty"`
	AlmacenRefID       int32  `json:",omitempty"`
	AlmacenRefCantidad int32  `json:",omitempty"`
	Fecha          int16  `json:",omitempty"`
	DocumentID            int64  `json:",omitempty"`
	ProductoID         int32  `json:",omitempty"`
	PresentacionID     int16  `json:",omitempty"`
	Cantidad           int32  `json:",omitempty"`
	AlmacenCantidad    int32  `json:",omitempty"`
	SubCantidad        int32  `json:",omitempty"`
	Tipo               int8   `json:",omitempty"`
	Created            int32  `json:",omitempty"`
	CreatedBy          int32  `json:",omitempty"`
}

type AlmacenMovimientoTable struct {
	db.TableStruct[AlmacenMovimientoTable, AlmacenMovimiento]
	EmpresaID          db.Col[AlmacenMovimientoTable, int32]
	ID                 db.Col[AlmacenMovimientoTable, int64]
	SKU                db.Col[AlmacenMovimientoTable, string]
	Lote               db.Col[AlmacenMovimientoTable, string]
	AlmacenID          db.Col[AlmacenMovimientoTable, int32]
	AlmacenRefID       db.Col[AlmacenMovimientoTable, int32]
	AlmacenRefCantidad db.Col[AlmacenMovimientoTable, int32]
	DocumentID         db.Col[AlmacenMovimientoTable, int64]
	ProductoID         db.Col[AlmacenMovimientoTable, int32]
	PresentacionID     db.Col[AlmacenMovimientoTable, int16]
	Cantidad           db.Col[AlmacenMovimientoTable, int32]
	AlmacenCantidad    db.Col[AlmacenMovimientoTable, int32]
	SubCantidad        db.Col[AlmacenMovimientoTable, int32]
	Tipo               db.Col[AlmacenMovimientoTable, int8]
	Created            db.Col[AlmacenMovimientoTable, int32]
	CreatedBy          db.Col[AlmacenMovimientoTable, int32]
	Fecha              db.Col[AlmacenMovimientoTable, int16]
}

func (e AlmacenMovimientoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "almacen_movimiento",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.AlmacenID.DecimalSize(5), e.Fecha.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Fecha,
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.SKU}, KeepPart: true},
			{Cols: []db.Coln{e.Lote}, KeepPart: true},
			{Cols: []db.Coln{e.AlmacenRefID, e.Created}, ConcatI64: []int8{9}},
		},
	}
}

type MovimientoInterno struct {
	ProductoID           int32
	PresentacionID       int16
		ReemplazarCantidad   bool
		Tipo int8
	SKU                  string
	Lote                 string
	AlmacenID            int32
	AlmacenDestinoID     int32
	Cantidad             int32
	SubCantidad          int32
	ModificarCantidad    int32
	ModificarSubCantidad int32
	DocumentID 					 int64
}

func (e *MovimientoInterno) GetAlmacenProductoID() string {
	return Concat62(e.AlmacenID, e.ProductoID, e.PresentacionID, e.SKU, e.Lote)
}
