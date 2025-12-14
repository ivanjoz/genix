package types

import "app/db2"

type Caja struct {
	db2.TableStruct[CajaTable, Caja]
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
	Updated   int32 `json:"upd,omitempty" db:"updated,view.1"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int32 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
}

type CajaTable struct {
	db2.TableStruct[CajaTable, Caja]
	EmpresaID    db2.Col[CajaTable, int32]
	ID           db2.Col[CajaTable, int32]
	Tipo         db2.Col[CajaTable, int32]
	SedeID       db2.Col[CajaTable, int32]
	Nombre       db2.Col[CajaTable, string]
	Descripcion  db2.Col[CajaTable, string]
	MonedaTipo   db2.Col[CajaTable, int8]
	CuadreFecha  db2.Col[CajaTable, int32]
	CuadreSaldo  db2.Col[CajaTable, int32]
	SaldoCurrent db2.Col[CajaTable, int32]
	Status       db2.Col[CajaTable, int8]
	Updated      db2.Col[CajaTable, int32]
	UpdatedBy    db2.Col[CajaTable, int32]
	Created      db2.Col[CajaTable, int32]
	CreatedBy    db2.Col[CajaTable, int32]
}

func (e CajaTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "cajas",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Status}, KeepPart: true},
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type CajaMovimiento struct {
	db2.TableStruct[CajaMovimientoTable, CajaMovimiento]
	EmpresaID  int32 `json:",omitempty" db:"empresa_id,pk"`
	ID         int64 `db:"id,pk"`
	CajaID     int32 `db:"caja_id"`
	CajaRefID  int32 `db:"caja_ref_id"`
	VentaID    int32 `json:",omitempty" db:"venta_id,view"`
	Tipo       int8  `json:",omitempty" db:"tipo"`
	SaldoFinal int32 `db:"saldo_final"`
	Monto      int32 `db:"monto"`
	Created    int32 `json:",omitempty" db:"created"`
	CreatedBy  int32 `json:",omitempty" db:"created_by,view.1"`
}

type CajaMovimientoTable struct {
	db2.TableStruct[CajaMovimientoTable, CajaMovimiento]
	EmpresaID  db2.Col[CajaMovimientoTable, int32]
	ID         db2.Col[CajaMovimientoTable, int64]
	CajaID     db2.Col[CajaMovimientoTable, int32]
	CajaRefID  db2.Col[CajaMovimientoTable, int32]
	VentaID    db2.Col[CajaMovimientoTable, int32]
	Tipo       db2.Col[CajaMovimientoTable, int8]
	SaldoFinal db2.Col[CajaMovimientoTable, int32]
	Monto      db2.Col[CajaMovimientoTable, int32]
	Created    db2.Col[CajaMovimientoTable, int32]
	CreatedBy  db2.Col[CajaMovimientoTable, int32]
}

func (e CajaMovimientoTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "caja_movimientos",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.VentaID}, KeepPart: true},
			{Cols: []db2.Coln{e.CreatedBy}, KeepPart: true},
		},
	}
}

type CajaCuadre struct {
	db2.TableStruct[CajaCuadreTable, CajaCuadre]
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

type CajaCuadreTable struct {
	db2.TableStruct[CajaCuadreTable, CajaCuadre]
	EmpresaID       db2.Col[CajaCuadreTable, int32]
	ID              db2.Col[CajaCuadreTable, int64]
	CajaID          db2.Col[CajaCuadreTable, int32]
	Tipo            db2.Col[CajaCuadreTable, int8]
	MovimientoID    db2.Col[CajaCuadreTable, int64]
	SaldoSistema    db2.Col[CajaCuadreTable, int32]
	SaldoReal       db2.Col[CajaCuadreTable, int32]
	SaldoDiferencia db2.Col[CajaCuadreTable, int32]
	Created         db2.Col[CajaCuadreTable, int32]
	CreatedBy       db2.Col[CajaCuadreTable, int32]
}

func (e CajaCuadreTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "caja_cuadre",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.CreatedBy}, KeepPart: true},
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
