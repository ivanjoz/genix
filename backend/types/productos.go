package types

type Increment struct {
	TAGS         `table:"sequences"`
	TableName    string `db:"name,pk"`
	CurrentValue int64  `db:"current_value,counter"`
}

type ProductoImagen struct {
	Name        string `ms:"n" json:"n"`
	Descripcion string `ms:"d" json:"d"`
}

type Producto struct {
	TAGS          `table:"productos"`
	EmpresaID     int32   `json:",omitempty" db:"empresa_id,pk"`
	ID            int32   `db:"id,pk"`
	TempID        int32   `json:"-" db:"temp_id"`
	Nombre        string  `db:"nombre"`
	Descripcion   string  `json:",omitempty" db:"descripcion"`
	GruposIDs     []int32 `json:",omitempty" db:"grupos_ids"`
	Params        []int8  `json:",omitempty" db:"params_ids"`
	Precio        float32 `json:",omitempty" db:"precio"`
	Descuento     float32 `json:",omitempty" db:"descuento"`
	PrecioFinal   float32 `json:",omitempty" db:"precio_final"`
	Peso          float32 `json:",omitempty" db:"peso"`
	Volumen       float32 `json:",omitempty" db:"volumen"`
	SbnCantidad   float32 `json:",omitempty" db:"sbn_cantidad"`
	SbnUnidad     string  `json:",omitempty" db:"sbn_unidad"`
	SbnPrecio     float32 `json:",omitempty" db:"sbn_precio"`
	SbnDescuento  float32 `json:",omitempty" db:"sbn_decuento"`
	SbnPreciFinal float32 `json:",omitempty" db:"sbn_precio_final"`

	Propiedades []ProductoPropiedad `json:",omitempty" db:"propiedades"`
	Images      []ProductoImagen    `json:",omitempty" db:"images"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int64 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
}

type ProductoPropiedad struct {
	ID      int16    `ms:"i"`
	Nombre  string   `ms:"n"`
	Options []string `ms:"o"`
}

type Almacen struct {
	TAGS        `table:"almacenes"`
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

type ProductoStock struct {
	TAGS             `table:"producto_stock"`
	EmpresaID        int32   `db:"empresa_id,pk"`
	ProductoID       int32   `db:"id,pk"`
	AlmacenID        int32   `db:"almacen_id"`
	LayoutID         int16   `db:"layout_id"`
	LayoutPosicionID int16   `db:"layout_posicion_id"`
	Stock            float32 `db:"stock"`
	SKU              string  `db:"sku"`
	Updated          int64   `json:"upd" db:"updated,view"`
	UpdatedBy        int32   `db:"updated_by"`
}

type AlmacenLayout struct {
	ID      int16                 `ms:"i"`
	Name    string                `ms:"n"`
	RowCant int8                  `ms:"r"`
	ColCant int8                  `ms:"c"`
	Bloques []AlmacenLayoutBloque `ms:"b"`
}

type AlmacenLayoutBloque struct {
	Row    int8   `json:"rw" ms:"r"`
	Column int8   `json:"co" ms:"c"`
	Name   string `json:"nm" ms:"n"`
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
	Ciudad      string
}

type AlmacenProducto struct {
	TAGS      `table:"almacen_producto"`
	EmpresaID int32 `json:",omitempty" db:"empresa_id,pk"`
	// [Almacen-ID] [status] [Producto-ID] [SKU] [Lote]
	ID             string  `db:"id,pk"`
	SKU            string  `json:",omitempty" db:"sku,view"`
	Lote           string  `json:",omitempty" db:"lote,view"`
	AlmacenID      int32   `json:",omitempty" db:"almacen_id"`
	ProductoID     int32   `json:",omitempty" db:"producto_id,view"`
	UpdatedBy      int32   `json:",omitempty" db:"updated_by"`
	Cantidad       int32   `json:",omitempty" db:"cantidad"`
	SubCantidad    int32   `json:",omitempty" db:"sub_cantidad"`
	CostoUn        float32 `json:",omitempty" db:"costo_un"`
	Status         int8    `json:"ss,omitempty" db:"status"`
	Updated        int64   `json:"upd,omitempty" db:"updated"`
	AlmacenUpdated int64   `json:"-" db:"sk_almacen_updated,view,exclude"`
}

func (e *AlmacenProducto) SelfParse() {
	if e.AlmacenID > 0 && e.Updated > 0 {
		e.AlmacenUpdated = int64(e.AlmacenID)*10_000_000_000 + e.Updated
	}
	e.ID = Concat62(e.AlmacenID, e.Status, e.ProductoID, e.SKU, e.Lote)
}

type AlmacenMovimiento struct {
	TAGS      `table:"almacen_movimiento"`
	EmpresaID int32 `json:",omitempty" db:"empresa_id,pk"`
	// [Almacen-ID] [Producto-ID] [Created] [SKU] [Random String]
	ID                    string `db:"id,pk"`
	SKU                   string `json:",omitempty" db:"sku,view"`
	Lote                  string `json:",omitempty" db:"lote,view"`
	AlmacenID             int32  `json:",omitempty" db:"almacen_id"`
	AlmacenOrigenID       int32  `json:",omitempty" db:"almacen_origen_id"`
	VentaID               int32  `json:",omitempty" db:"venta_id"`
	ProductoID            int32  `json:",omitempty" db:"producto_id"`
	Cantidad              int32  `json:",omitempty" db:"cantidad"`
	AlmacenCantidad       int32  `json:",omitempty" db:"almacen_cantidad"`
	AlmacenOrigenCantidad int32  `json:",omitempty" db:"almacen_origen_cantidad"`
	SubCantidad           int32  `json:",omitempty" db:"sub_cantidad"`
	Tipo                  int8   `json:",omitempty" db:"tipo"`
	Created               int64  `json:",omitempty" db:"created"`
	ProductoCreated       int64  `json:"-" db:"sk_producto_created,view,exclude"`
	AlmacenCreated        int64  `json:"-" db:"sk_almacen_created,view,exclude"`
	AlmacenOrigenCreated  int64  `json:"-" db:"sk_almacen_origen_created,view,exclude"`
}

func (e *AlmacenMovimiento) SelfParse() {
	e.AlmacenCreated = ConcatInt64(int64(e.AlmacenID), e.Created)
	e.AlmacenOrigenCreated = ConcatInt64(int64(e.AlmacenOrigenID), e.Created)
	e.ProductoCreated = ConcatInt64(int64(e.ProductoID), e.Created)
}

func (e *AlmacenMovimiento) AssignID() {
	rnd := MakeRandomBase36String(4)
	e.ID = Concat62(e.AlmacenID, e.ProductoID, e.Created, e.SKU, e.Lote, rnd)
}
