package types

import "app/db"

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

func (e Caja) EmpresaID_() db.CoI32    { return db.CoI32{"empresa_id"} }
func (e Caja) ID_() db.CoI32           { return db.CoI32{"id"} }
func (e Caja) Tipo_() db.CoI32         { return db.CoI32{"tipo"} }
func (e Caja) SedeID_() db.CoI32       { return db.CoI32{"sede_id"} }
func (e Caja) Nombre_() db.CoStr       { return db.CoStr{"nombre"} }
func (e Caja) Descripcion_() db.CoStr  { return db.CoStr{"descripcion"} }
func (e Caja) MonedaTipo_() db.CoI8    { return db.CoI8{"moneda"} }
func (e Caja) CuadreFecha_() db.CoI32  { return db.CoI32{"cuadre_fecha"} }
func (e Caja) CuadreSaldo_() db.CoI32  { return db.CoI32{"cuadre_saldo"} }
func (e Caja) SaldoCurrent_() db.CoI32 { return db.CoI32{"saldo_current"} }
func (e Caja) Status_() db.CoI8        { return db.CoI8{"status"} }
func (e Caja) Updated_() db.CoI32      { return db.CoI32{"updated"} }
func (e Caja) UpdatedBy_() db.CoI32    { return db.CoI32{"updated_by"} }
func (e Caja) Created_() db.CoI32      { return db.CoI32{"created"} }
func (e Caja) CreatedBy_() db.CoI32    { return db.CoI32{"created_by"} }

func (e Caja) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "cajas",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.Updated_()}, KeepPart: true},
		},
	}
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

type CajaMovimientoTable struct {
	db.GetSchemaTest1[CajaMovimiento]
	EmpresaID db.Col[int32]
	CajaID    db.Colx[CajaMovimientoTable, int32]
}

func (e CajaMovimientoTable) MakeTable() *CajaMovimientoTable {
	return &CajaMovimientoTable{}
}

type cjm = CajaMovimiento

func (e cjm) EmpresaID_() db.CoI32  { return db.CoI32{"empresa_id"} }
func (e cjm) ID_() db.CoI64         { return db.CoI64{"id"} }
func (e cjm) VentaID_() db.CoI32    { return db.CoI32{"venta_id"} }
func (e cjm) CajaID_() db.CoI32     { return db.CoI32{"caja_id"} }
func (e cjm) CajaRefID_() db.CoI32  { return db.CoI32{"caja_ref_id"} }
func (e cjm) Tipo_() db.CoI32       { return db.CoI32{"tipo"} }
func (e cjm) SaldoFinal_() db.CoI32 { return db.CoI32{"saldo_final"} }
func (e cjm) Monto_() db.CoI32      { return db.CoI32{"monto"} }
func (e cjm) Created_() db.CoI32    { return db.CoI32{"created"} }
func (e cjm) CreatedBy_() db.CoI32  { return db.CoI32{"created_by"} }

func (e cjm) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_movimientos",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.VentaID_()}, KeepPart: true},
			{Cols: []db.Coln{e.CreatedBy_()}, KeepPart: true},
		},
	}
}

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

func (e CajaCuadre) EmpresaID_() db.CoI32       { return db.CoI32{"empresa_id"} }
func (e CajaCuadre) ID_() db.CoI64              { return db.CoI64{"id"} }
func (e CajaCuadre) CajaID_() db.CoI32          { return db.CoI32{"caja_id"} }
func (e CajaCuadre) Tipo_() db.CoI8             { return db.CoI8{"tipo"} }
func (e CajaCuadre) SaldoSistema_() db.CoI32    { return db.CoI32{"saldo_sistema"} }
func (e CajaCuadre) SaldoReal_() db.CoI32       { return db.CoI32{"saldo_real"} }
func (e CajaCuadre) SaldoDiferencia_() db.CoI32 { return db.CoI32{"saldo_diferencia"} }
func (e CajaCuadre) Created_() db.CoI32         { return db.CoI32{"created"} }
func (e CajaCuadre) CreatedBy_() db.CoI32       { return db.CoI32{"created_by"} }

func (e CajaCuadre) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_cuadre",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.CreatedBy_()}, KeepPart: true},
		},
	}
}

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
