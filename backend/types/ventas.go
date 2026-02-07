package types

import "app/db"

type Caja struct {
	db.TableStruct[CajaTable, Caja]
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
	db.TableStruct[CajaTable, Caja]
	EmpresaID    db.Col[CajaTable, int32]
	ID           db.Col[CajaTable, int32]
	Tipo         db.Col[CajaTable, int32]
	SedeID       db.Col[CajaTable, int32]
	Nombre       db.Col[CajaTable, string]
	Descripcion  db.Col[CajaTable, string]
	MonedaTipo   db.Col[CajaTable, int8]
	CuadreFecha  db.Col[CajaTable, int32]
	CuadreSaldo  db.Col[CajaTable, int32]
	SaldoCurrent db.Col[CajaTable, int32]
	Status       db.Col[CajaTable, int8]
	Updated      db.Col[CajaTable, int32]
	UpdatedBy    db.Col[CajaTable, int32]
	Created      db.Col[CajaTable, int32]
	CreatedBy    db.Col[CajaTable, int32]
}

func (e CajaTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "cajas",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID},
		Views: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type CajaMovimiento struct {
	db.TableStruct[CajaMovimientoTable, CajaMovimiento]
	EmpresaID  int32 `json:",omitempty"`
	ID         int64 
	CajaID     int32 
	CajaRefID  int32 
	VentaID    int32 `json:",omitempty"`
	Fecha int16
	Tipo       int8  `json:",omitempty"`
	SaldoFinal int32 
	Monto      int32 
	Created    int32 `json:",omitempty"`
	CreatedBy  int32 `json:",omitempty"`
}

type CajaMovimientoTable struct {
	db.TableStruct[CajaMovimientoTable, CajaMovimiento]
	EmpresaID  db.Col[CajaMovimientoTable, int32]
	ID         db.Col[CajaMovimientoTable, int64]
	CajaID     db.Col[CajaMovimientoTable, int32]
	CajaRefID  db.Col[CajaMovimientoTable, int32]
	VentaID    db.Col[CajaMovimientoTable, int32]	
	Fecha  db.Col[CajaMovimientoTable, int16]
	Tipo       db.Col[CajaMovimientoTable, int8]
	SaldoFinal db.Col[CajaMovimientoTable, int32]
	Monto      db.Col[CajaMovimientoTable, int32]
	Created    db.Col[CajaMovimientoTable, int32]
	CreatedBy  db.Col[CajaMovimientoTable, int32]
}

func (e CajaMovimientoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_movimientos",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			e.CajaID.DecimalSize(5), e.Fecha.DecimalSize(5), e.Autoincrement(3),
		},
		Views: []db.View{
			{Cols: []db.Coln{e.VentaID}, KeepPart: true},
			{Cols: []db.Coln{e.CreatedBy}, KeepPart: true},
		},
	}
}

type CajaCuadre struct {
	db.TableStruct[CajaCuadreTable, CajaCuadre]
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
	db.TableStruct[CajaCuadreTable, CajaCuadre]
	EmpresaID       db.Col[CajaCuadreTable, int32]
	ID              db.Col[CajaCuadreTable, int64]
	CajaID          db.Col[CajaCuadreTable, int32]
	Tipo            db.Col[CajaCuadreTable, int8]
	MovimientoID    db.Col[CajaCuadreTable, int64]
	SaldoSistema    db.Col[CajaCuadreTable, int32]
	SaldoReal       db.Col[CajaCuadreTable, int32]
	SaldoDiferencia db.Col[CajaCuadreTable, int32]
	Created         db.Col[CajaCuadreTable, int32]
	CreatedBy       db.Col[CajaCuadreTable, int32]
}

func (e CajaCuadreTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_cuadre",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		Views: []db.View{
			{Cols: []db.Coln{e.CreatedBy}, KeepPart: true},
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
