package types

import "app/db"


type Sale struct {
	db.TableStruct[SaleTable, Sale]
	EmpresaID int32 `json:",omitempty"`
	Fecha int16   `json:",omitempty"`
	AlmacenID int32   `json:",omitempty"`
	ID             int64
	
	//Tabla: Following slices must be same size
	DetailProductsIDs []int32 `json:",omitempty"`
	DetailPrices []int32 `json:",omitempty"`
	DetailQuantities []int32 `json:",omitempty"`
	
	InvoiceAmount int32 `json:",omitempty"`
	SubTotal int32 `json:",omitempty"`
	DebtAmount int32 `json:",omitempty"`
	DeliveryStatus int8 `json:",omitempty"`
	CajaID_ int32  `json:",omitempty"`
	
	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ProcessesIncluded_ []int8 `json:",omitempty"`
	Created        int32   `json:",omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Status         int8    `json:"ss,omitempty"`
}

type SaleTable struct {
	db.TableStruct[SaleTable, Sale]
	EmpresaID          db.Col[SaleTable, int32]
	ID                 db.Col[SaleTable, int64]
		Fecha  db.Col[SaleTable, int16]
				AlmacenID  db.Col[SaleTable, int32]
								Updated  db.Col[SaleTable, int32]
}

func (e SaleTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "sales",
		Partition:       e.EmpresaID,
		Keys:            []db.Coln{e.ID.Autoincrement(3)},
		AutoincrementPart: e.Fecha,
		Views: []db.View{
			{ Cols: []db.Coln{ e.Fecha, e.Updated }, KeepPart: true },
		},
	}
}
