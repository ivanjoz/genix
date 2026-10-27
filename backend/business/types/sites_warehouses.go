package types

import "app/db"

type Warehouse struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   int32             `cb:"CompanyID,1"`
	ID          int32             `cb:"ID,2"`
	SiteID      int32             `cb:"SiteID,3"`
	Name        string            `cb:"Name,4"`
	Description string            `cb:"Description,5"`
	Layout      []WarehouseLayout `cb:"Layout,6"`
	// General properties
	Status  int8  `json:"ss,omitempty" db:"status" cb:"ss,7"`
	Updated int32 `json:"upd,omitempty" db:"updated" cb:"upd,8"`
	Version int16 `json:"ver,omitempty" cb:"ver,9"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt,omitempty" cb:"mdt,10"`
	UpdatedBy int32   `json:",omitempty" db:"updated_by" cb:"UpdatedBy,11"`
	Created   int32   `json:",omitempty" db:"created" cb:"Created,12"`
	CreatedBy int32   `json:",omitempty" db:"created_by" cb:"CreatedBy,13"`
	City      string  `json:",omitempty" cb:"City,14"`
}

type WarehouseTable struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   db.Col[*WarehouseTable, int32]
	ID          db.Col[*WarehouseTable, int32]
	SiteID      db.Col[*WarehouseTable, int32]
	Name        db.Col[*WarehouseTable, string]
	Description db.Col[*WarehouseTable, string]
	Layout      db.Col[*WarehouseTable, []WarehouseLayout]
	Status      db.Col[*WarehouseTable, int8]
	Updated     db.Col[*WarehouseTable, int32]
	Version     db.Col[*WarehouseTable, int16]
	UpdatedBy   db.Col[*WarehouseTable, int32]
	Created     db.Col[*WarehouseTable, int32]
	CreatedBy   db.Col[*WarehouseTable, int32]
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
	ID      int16                  `ms:"i" json:"id,omitempty" cb:"id,1"`
	Name    string                 `ms:"n" json:"nm,omitempty" cb:"nm,2"`
	RowCant int8                   `ms:"r" json:"rc,omitempty" cb:"rc,3"`
	ColCant int8                   `ms:"c" json:"cc,omitempty" cb:"cc,4"`
	Bloques []WarehouseLayoutBlock `ms:"b" json:"bl,omitempty" cb:"bl,5"`
}

type WarehouseLayoutBlock struct {
	Row    int8   `json:"rw" ms:"r" cb:"rw,1"`
	Column int8   `json:"co" ms:"c" cb:"co,2"`
	Name   string `json:"nm" ms:"n" cb:"nm,3"`
}

type Site struct {
	db.TableStruct[SiteTable, Site]
	CompanyID   int32  `db:"empresa_id,pk" cb:"CompanyID,1"`
	ID          int32  `db:"id,pk" cb:"ID,2"`
	Name        string `db:"nombre" cb:"Name,3"`
	Description string `db:"descripcion" cb:"Description,4"`
	Address     string `db:"direccion" cb:"Address,5"`
	CityID      int32  `db:"pais_ciudad_id" cb:"CityID,6"`
	City        string `json:",omitempty" db:"-" cb:"City,7"`
	Status      int8   `json:"ss,omitempty" db:"status" cb:"ss,8"`
	Updated     int32  `json:"upd,omitempty" db:"updated" cb:"upd,9"`
	Version     int16  `json:"ver,omitempty" cb:"ver,10"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt,omitempty" cb:"mdt,11"`
	UpdatedBy int32   `json:",omitempty" db:"updated_by" cb:"UpdatedBy,12"`
	Created   int32   `json:",omitempty" db:"created" cb:"Created,13"`
	CreatedBy int32   `json:",omitempty" db:"created_by" cb:"CreatedBy,14"`
}

type SiteTable struct {
	db.TableStruct[SiteTable, Site]
	CompanyID   db.Col[*SiteTable, int32]
	ID          db.Col[*SiteTable, int32]
	Name        db.Col[*SiteTable, string]
	Description db.Col[*SiteTable, string]
	Address     db.Col[*SiteTable, string]
	CityID      db.Col[*SiteTable, int32] `db:"pais_ciudad_id"`
	Status      db.Col[*SiteTable, int8]
	Updated     db.Col[*SiteTable, int32]
	Version     db.Col[*SiteTable, int16]
	UpdatedBy   db.Col[*SiteTable, int32]
	Created     db.Col[*SiteTable, int32]
	CreatedBy   db.Col[*SiteTable, int32]
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
