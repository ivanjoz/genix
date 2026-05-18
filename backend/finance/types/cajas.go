package types

import "app/db"

type CashBank struct {
	db.TableStruct[CashBankTable, CashBank]
	CompanyID    int32
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

type CashBankTable struct {
	db.TableStruct[CashBankTable, CashBank]
	CompanyID    db.Col[CashBankTable, int32]
	ID           db.Col[CashBankTable, int32]
	Tipo         db.Col[CashBankTable, int32]
	SedeID       db.Col[CashBankTable, int32]
	Nombre       db.Col[CashBankTable, string]
	Descripcion  db.Col[CashBankTable, string]
	MonedaTipo   db.Col[CashBankTable, int8]
	CuadreFecha  db.Col[CashBankTable, int32]
	CuadreSaldo  db.Col[CashBankTable, int32]
	SaldoCurrent db.Col[CashBankTable, int32]
	Status       db.Col[CashBankTable, int8]
	Updated      db.Col[CashBankTable, int32]
	UpdatedBy    db.Col[CashBankTable, int32]
	Created      db.Col[CashBankTable, int32]
	CreatedBy    db.Col[CashBankTable, int32]
}

func (e CashBankTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "cajas",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type CashBankMovement struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID   int32 `json:",omitempty"`
	ID          int64
	CajaID      int32
	CajaRefID   int32
	DocumentoID int64 `json:",omitempty"`
	Date       int16
	Tipo        int8 `json:",omitempty"`
	SaldoFinal  int32
	Monto       int32
	Created     int32 `json:",omitempty"`
	CreatedBy   int32 `json:",omitempty"`
}

type CashBankMovementTable struct {
	db.TableStruct[CashBankMovementTable, CashBankMovement]
	CompanyID   db.Col[CashBankMovementTable, int32]
	ID          db.Col[CashBankMovementTable, int64]
	CajaID      db.Col[CashBankMovementTable, int32]
	CajaRefID   db.Col[CashBankMovementTable, int32]
	DocumentoID db.Col[CashBankMovementTable, int64]
	Date       db.Col[CashBankMovementTable, int16]
	Tipo        db.Col[CashBankMovementTable, int8]
	SaldoFinal  db.Col[CashBankMovementTable, int32]
	Monto       db.Col[CashBankMovementTable, int32]
	Created     db.Col[CashBankMovementTable, int32]
	CreatedBy   db.Col[CashBankMovementTable, int32]
}

func (e CashBankMovementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_movimientos",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		KeyIntPacking: []db.Coln{
			//TODO: decrease to e.Autoincrement(2) in the future
			e.CajaID.DecimalSize(5), e.Date.DecimalSize(5), e.Autoincrement(3),
		},
		AutoincrementPart: e.Date,
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.DocumentoID}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.CreatedBy}},
		},
	}
}

type CashReconciliation struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID       int32 `json:",omitempty"`
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

type CashReconciliationTable struct {
	db.TableStruct[CashReconciliationTable, CashReconciliation]
	CompanyID       db.Col[CashReconciliationTable, int32]
	ID              db.Col[CashReconciliationTable, int64]
	CajaID          db.Col[CashReconciliationTable, int32]
	Tipo            db.Col[CashReconciliationTable, int8]
	MovimientoID    db.Col[CashReconciliationTable, int64]
	SaldoSistema    db.Col[CashReconciliationTable, int32]
	SaldoReal       db.Col[CashReconciliationTable, int32]
	SaldoDiferencia db.Col[CashReconciliationTable, int32]
	Created         db.Col[CashReconciliationTable, int32]
	CreatedBy       db.Col[CashReconciliationTable, int32]
}

func (e CashReconciliationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "caja_cuadre",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.CreatedBy}, KeepPart: true},
		},
	}
}

type InternalCashMovement struct {
	CajaID     int32
	CajaRefID  int32
	DocumentID int64
	Tipo       int8
	Monto      int32
	SaldoFinal int32 // Opcional, si es 0 se calculará
}

type SaleProduct struct {
	ProductID int32 `cbor:"1,keyasint"`
	Cantidad   int32 `cbor:"2,keyasint"`
	Monto      int32 `cbor:"3,keyasint"`
}
