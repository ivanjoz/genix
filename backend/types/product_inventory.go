package types

import "app/db"

type ProductInventory struct {
	db.TableStruct[ProductInventoryTable, ProductInventory]
	EmpresaID   int32   `json:"empresa_id,omitempty" db:"empresa_id"`
	Status      int8    `json:"ss,omitempty" db:"status"`
	Updated     int64   `json:"upd,omitempty" db:"updated"`
	UpdatedBy   int32   `json:"updated_by,omitempty" db:"updated_by"`
	Name        string  `json:"name,omitempty" db:"name"`
	Description string  `json:"description,omitempty" db:"description"`
	Stock       int32   `json:"stock,omitempty" db:"stock"`
	Price       float64 `json:"price,omitempty" db:"price"`
	Sku         string  `json:"sku,omitempty" db:"sku,pk"`
}

type ProductInventoryTable struct {
	db.TableStruct[ProductInventoryTable, ProductInventory]
	EmpresaID   db.Col[ProductInventoryTable, int32]
	Status      db.Col[ProductInventoryTable, int8]
	Updated     db.Col[ProductInventoryTable, int64]
	UpdatedBy   db.Col[ProductInventoryTable, int32]
	Name        db.Col[ProductInventoryTable, string]
	Description db.Col[ProductInventoryTable, string]
	Stock       db.Col[ProductInventoryTable, int32]
	Price       db.Col[ProductInventoryTable, float64]
	Sku         db.Col[ProductInventoryTable, string]
}

func (e ProductInventoryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "product_inventory",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.Sku},
	}
}
