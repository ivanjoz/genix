package types

type Caja struct {
	TAGS         `table:"cajas"`
	EmpresaID    int32  `db:"empresa_id,pk"`
	ID           int32  `db:"id,pk"`
	Tipo         int32  `db:"tipo"`
	SedeID       int32  `db:"sede_id"`
	Nombre       string `db:"nombre"`
	Descripcion  string `db:"descripcion"`
	MonedaTipo   int8   `db:"moneda"`
	CuadreFecha  int32  `db:"cuadre_fecha"`
	CuadreSaldo  int32  `db:"cuadre_saldo"`
	SaldoCurrent int32  `db:"saldo_current"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int32 `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int32 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
}

type CajaMovimiento struct {
	TAGS       `table:"caja_movimientos"`
	EmpresaID  int32 `json:",omitempty" db:"empresa_id,pk"`
	ID         int64 `db:"id,pk"`
	CajaID     int32 `db:"caja_id"`
	CajaRefID  int32 `db:"caja_ref_id"`
	VentaID    int32 `json:",omitempty" db:"venta_id,view"`
	Tipo       int8  `json:",omitempty" db:"tipo"`
	SaldoFinal int32 `db:"saldo_final"`
	Monto      int32 `db:"monto"`
	Created    int32 `json:",omitempty" db:"created"`
	CreatedBy  int32 `json:",omitempty" db:"created_by,view"`
}

/*
func (e *CajaMovimiento) MakeID(unixTimeMill int64) int64 {
	return int64(e.CajaID)*10_000_000_000_000 + unixTimeMill
}

func (e *CajaMovimiento) SetID(unixTimeMill int64) {
	e.ID = int64(e.CajaID)*10_000_000_000_000 + unixTimeMill
}
*/

type CajaCuadre struct {
	TAGS            `table:"caja_cuadre"`
	EmpresaID       int32 `json:",omitempty" db:"empresa_id,pk"`
	ID              int64 `json:",omitempty" db:"id,pk"`
	CajaID          int32 `json:",omitempty" db:"caja_id"`
	Tipo            int8  `json:",omitempty" db:"tipo"`
	MovimientoID    int64 `json:",omitempty" db:"movimiento_id"`
	SaldoSistema    int32 `json:",omitempty" db:"saldo_sistema"`
	SaldoReal       int32 `json:",omitempty" db:"saldo_real"`
	SaldoDiferencia int32 `json:",omitempty" db:"saldo_diferencia"`
	Created         int32 `json:",omitempty" db:"created"`
	CreatedBy       int32 `json:",omitempty" db:"created_by,view"`
}

/*
func (e *CajaCuadre) SetID(unixTimeMill int64) {
	e.ID = int64(e.CajaID)*10_000_000_000_00 + unixTimeMill
}
*/

type VentaProducto struct {
	ProductoID int32 `cbor:"1,keyasint"`
	Cantidad   int32 `cbor:"2,keyasint"`
	Monto      int32 `cbor:"3,keyasint"`
}

type Venta struct {
	TAGS           `table:"ventas"`
	EmpresaID      int32 `db:"empresa_id,pk"`
	ID             int64 `db:"id,pk"` // [Fecha Unix + Autoincrement]
	Monto          int32 `db:"monto"`
	MontoPorCobrar int32 `db:"monto_por_cobrar"`
	ClienteID      int32 `db:"cliente_id"`
	Tipo           int8  `db:"tipo"`
	Status         int8  `db:"ss"`
	EntregaStatus  int8  `db:"entrega_status"`
	Productos      int32 `db:"productos"`
	Created        int32 `db:"created"`
	CreatedBY      int32 `db:"created_by"`
}
