package types

import "app/db"

type Warehouse struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   int32
	ID          int32
	SiteID      int32
	Name        string
	Description string
	Layout      []WarehouseLayout
	// General properties
	Status         int8   `json:"ss,omitempty" db:"status"`
	Updated        int32  `json:"upd,omitempty" db:"updated"`
	UpdatedVersion int32  `json:"upv,omitempty"`
	UpdatedBy      int32  `json:",omitempty" db:"updated_by"`
	Created        int32  `json:",omitempty" db:"created"`
	CreatedBy      int32  `json:",omitempty" db:"created_by"`
	City           string `json:",omitempty"`
}

type WarehouseTable struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID      db.Col[*WarehouseTable, int32]
	ID             db.Col[*WarehouseTable, int32]
	SiteID         db.Col[*WarehouseTable, int32]
	Name           db.Col[*WarehouseTable, string]
	Description    db.Col[*WarehouseTable, string]
	Layout         db.Col[*WarehouseTable, []WarehouseLayout]
	Status         db.Col[*WarehouseTable, int8]
	Updated        db.Col[*WarehouseTable, int32]
	UpdatedVersion db.Col[*WarehouseTable, int32]
	UpdatedBy      db.Col[*WarehouseTable, int32]
	Created        db.Col[*WarehouseTable, int32]
	CreatedBy      db.Col[*WarehouseTable, int32]
}

func (e WarehouseTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           39,
		Name:         "warehouses",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

type WarehouseLayout struct {
	ID      int16                  `ms:"i" json:"id,omitempty"`
	Name    string                 `ms:"n" json:"nm,omitempty"`
	RowCant int8                   `ms:"r" json:"rc,omitempty"`
	ColCant int8                   `ms:"c" json:"cc,omitempty"`
	Bloques []WarehouseLayoutBlock `ms:"b" json:"bl,omitempty"`
}

type WarehouseLayoutBlock struct {
	Row    int8   `json:"rw" ms:"r"`
	Column int8   `json:"co" ms:"c"`
	Name   string `json:"nm" ms:"n"`
}

type Site struct {
	db.TableStruct[SiteTable, Site]
	CompanyID      int32  `db:"empresa_id,pk"`
	ID             int32  `db:"id,pk"`
	Name           string `db:"nombre"`
	Description    string `db:"descripcion"`
	Address        string `db:"direccion"`
	CityID         int32  `db:"pais_ciudad_id"`
	City           string `json:",omitempty" db:"-"`
	Status         int8   `json:"ss,omitempty" db:"status"`
	Updated        int32  `json:"upd,omitempty" db:"updated"`
	UpdatedVersion int32  `json:"upv,omitempty"`
	UpdatedBy      int32  `json:",omitempty" db:"updated_by"`
	Created        int32  `json:",omitempty" db:"created"`
	CreatedBy      int32  `json:",omitempty" db:"created_by"`
}

type SiteTable struct {
	db.TableStruct[SiteTable, Site]
	CompanyID      db.Col[*SiteTable, int32]
	ID             db.Col[*SiteTable, int32]
	Name           db.Col[*SiteTable, string]
	Description    db.Col[*SiteTable, string]
	Address        db.Col[*SiteTable, string]
	CityID         db.Col[*SiteTable, int32] `db:"pais_ciudad_id"`
	Status         db.Col[*SiteTable, int8]
	Updated        db.Col[*SiteTable, int32]
	UpdatedVersion db.Col[*SiteTable, int32]
	UpdatedBy      db.Col[*SiteTable, int32]
	Created        db.Col[*SiteTable, int32]
	CreatedBy      db.Col[*SiteTable, int32]
}

func (e SiteTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           31,
		Name:         "sites",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}
