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
	CategoriasIDs []int32 `json:",omitempty" db:"categorias_ids"`
	Params        []int8  `json:",omitempty" db:"params_ids"`
	Precio        int32   `json:",omitempty" db:"precio"`
	Descuento     float32 `json:",omitempty" db:"descuento"`
	PrecioFinal   int32   `json:",omitempty" db:"precio_final"`
	Peso          float32 `json:",omitempty" db:"peso"`
	Volumen       float32 `json:",omitempty" db:"volumen"`
	SbnCantidad   int32   `json:",omitempty" db:"sbn_cantidad"`
	SbnUnidad     string  `json:",omitempty" db:"sbn_unidad"`
	SbnPrecio     int32   `json:",omitempty" db:"sbn_precio"`
	SbnDescuento  float32 `json:",omitempty" db:"sbn_decuento"`
	SbnPreciFinal int32   `json:",omitempty" db:"sbn_precio_final"`

	Propiedades []ProductoPropiedades `json:",omitempty" db:"propiedades"`
	Images      []ProductoImagen      `json:",omitempty" db:"images"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int64 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
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
	ID          string  `db:"id,pk"`
	SKU         string  `json:",omitempty" db:"sku,view"`
	Lote        string  `json:",omitempty" db:"lote,view"`
	AlmacenID   int32   `json:",omitempty" db:"almacen_id,view.1"`
	ProductoID  int32   `json:",omitempty" db:"producto_id,view"`
	UpdatedBy   int32   `json:",omitempty" db:"updated_by"`
	Cantidad    int32   `json:",omitempty" db:"cantidad"`
	SubCantidad int32   `json:",omitempty" db:"sub_cantidad"`
	CostoUn     float32 `json:",omitempty" db:"costo_un"`
	Updated     int32   `json:"upd,omitempty" db:"updated,view.1"`
	Status      int8    `json:"ss,omitempty" db:"status"`
}

func (e *AlmacenProducto) SelfParse() {
	e.ID = Concat62(e.AlmacenID, e.Status, e.ProductoID, e.SKU, e.Lote)
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
