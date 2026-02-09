package types

import "app/db"

type Caja struct {
	db.TableStruct[CajaTable, Caja]
	EmpresaID    int32 
	ID           int32 
	Tipo         int32 
	SedeID       int32  
	Nombre       string 
	Descripcion  string 
	MonedaTipo   int8  
	CuadreFecha  int32 
	CuadreSaldo  int32  
	SaldoCurrent int32 
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
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
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		ViewsDeprecated: []db.View{
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
	DocumentoID    int64 `json:",omitempty"`
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
	DocumentoID    db.Col[CajaMovimientoTable, int64]	
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
		AutoincrementPart: e.Fecha,
		LocalIndexes: []db.Coln{ e.DocumentoID, e.CreatedBy },
	}
}

type CajaCuadre struct {
	db.TableStruct[CajaCuadreTable, CajaCuadre]
	EmpresaID       int32 `json:",omitempty"`
	ID              int64 `json:",omitempty"`
	CajaID          int32 `json:",omitempty"`
	Tipo            int8  `json:",omitempty"`
	MovimientoID    int64 `json:",omitempty"`
	SaldoSistema    int32 `json:",omitempty"`
	SaldoReal       int32 `json:",omitempty"`
	SaldoDiferencia int32 `json:",omitempty"`
	Created         int32 `json:",omitempty"`
	CreatedBy       int32 `json:",omitempty"`
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
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.CreatedBy}, KeepPart: true},
		},
	}
}

type CajaMovimientoInterno struct {
	CajaID     int32
	CajaRefID  int32
		DocumentID    int64
	Tipo       int8
	Monto      int32
	SaldoFinal int32 // Opcional, si es 0 se calculará
}

type VentaProducto struct {
	ProductoID int32 `cbor:"1,keyasint"`
	Cantidad   int32 `cbor:"2,keyasint"`
	Monto      int32 `cbor:"3,keyasint"`
}
