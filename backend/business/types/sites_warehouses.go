package types

import (
	"app/core"
	"app/db"
	"fmt"
	"math"
)

type Warehouse struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID   int32             `cb:"1"`
	ID          int32             `cb:"2"`
	SiteID      int32             `cb:"3"`
	Name        string            `cb:"4"`
	Description string            `cb:"5"`
	Layout      []WarehouseLayout `cb:"6"`
	// General properties
	Status  int8  `json:"ss" cb:"ss,7"`
	Updated int32 `json:"upd" cb:"upd,8"`
	Version int16 `json:"ver" cb:"ver,9"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,10"`
	UpdatedBy int32   `cb:"11"`
	Created   int32   `cb:"12"`
	CreatedBy int32   `cb:"13"`
	City      string  `cb:"14"`
	// ExcludeFromOnlineSales keeps this warehouse's stock out of what the web store can sell.
	ExcludeFromOnlineSales bool `cb:"15"`
	// SubcontractorID makes this "material at workshop X" (docs/MANUFACTURING_PLAN.md §8): the warehouse
	// a subcontracted production order ships its supplies to and consumes them from. A provider
	// (client_provider Type 2) has at most one. Its stock is never sold or shown on the web.
	SubcontractorID int32 `cb:"16"`
}

// IsOpenToOnlineSales: an active warehouse whose stock the web store can sell.
func (e *Warehouse) IsOpenToOnlineSales() bool {
	return e.Status > 0 && !e.ExcludeFromOnlineSales
}

// OnlineWarehouseIDs are the warehouses whose stock the web store sells.
func OnlineWarehouseIDs(warehouses []Warehouse) []int32 {
	warehouseIDs := []int32{}
	for index := range warehouses {
		if warehouses[index].IsOpenToOnlineSales() {
			warehouseIDs = append(warehouseIDs, warehouses[index].ID)
		}
	}
	return warehouseIDs
}

// SiteIDsOfWarehouses is the site of each warehouse, as the int16 the stock ledger and the sales stamp
// for the per-site DataFrames. A warehouse that doesn't exist, or whose site doesn't fit an int16,
// refuses the whole lookup: a record must never be written under a wrong or missing site.
func SiteIDsOfWarehouses(companyID int32, warehouseIDs []int32) (map[int32]int16, error) {
	warehouses := []Warehouse{}
	query := db.Query(&warehouses)
	query.Select(query.ID, query.SiteID).CompanyID.Equals(companyID).ID.In(warehouseIDs...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las sedes de los almacenes:", err)
	}
	siteIDByWarehouseID := map[int32]int16{}
	for _, warehouse := range warehouses {
		if warehouse.SiteID <= 0 || warehouse.SiteID > math.MaxInt16 {
			return nil, core.Err(fmt.Sprintf("El almacén %v tiene una sede inválida (%v).", warehouse.ID, warehouse.SiteID))
		}
		siteIDByWarehouseID[warehouse.ID] = int16(warehouse.SiteID)
	}
	for _, warehouseID := range warehouseIDs {
		if _, isFound := siteIDByWarehouseID[warehouseID]; !isFound {
			return nil, core.Err(fmt.Sprintf("No se encontró el almacén %v.", warehouseID))
		}
	}
	return siteIDByWarehouseID, nil
}

// ValidateSiteID refuses a site the company does not have or has retired. 0 (no site, the whole
// company) is always valid.
func ValidateSiteID(companyID int32, siteID int16) error {
	if siteID == 0 {
		return nil
	}
	if siteID < 0 {
		return core.Err("La sede no es válida.")
	}
	sites := []Site{}
	query := db.Query(&sites)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.Equals(int32(siteID))
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener la sede:", err)
	}
	if len(sites) == 0 || sites[0].Status == 0 {
		return core.Err(fmt.Sprintf("La sede %v no existe o está inactiva.", siteID))
	}
	return nil
}

// FirstOnlineWarehouseID is the lowest-ID warehouse open to online sales, 0 when there is none: the
// one a web order's units are reserved on until it is delivered.
func FirstOnlineWarehouseID(warehouses []Warehouse) int32 {
	firstWarehouseID := int32(0)
	for index := range warehouses {
		warehouse := &warehouses[index]
		if warehouse.IsOpenToOnlineSales() && (firstWarehouseID == 0 || warehouse.ID < firstWarehouseID) {
			firstWarehouseID = warehouse.ID
		}
	}
	return firstWarehouseID
}

type WarehouseTable struct {
	db.TableStruct[WarehouseTable, Warehouse]
	CompanyID              db.Col[*WarehouseTable, int32]
	ID                     db.Col[*WarehouseTable, int32]
	SiteID                 db.Col[*WarehouseTable, int32]
	Name                   db.Col[*WarehouseTable, string]
	Description            db.Col[*WarehouseTable, string]
	Layout                 db.Col[*WarehouseTable, []WarehouseLayout]
	Status                 db.Col[*WarehouseTable, int8]
	Updated                db.Col[*WarehouseTable, int32]
	Version                db.Col[*WarehouseTable, int16]
	UpdatedBy              db.Col[*WarehouseTable, int32]
	Created                db.Col[*WarehouseTable, int32]
	CreatedBy              db.Col[*WarehouseTable, int32]
	ExcludeFromOnlineSales db.Col[*WarehouseTable, bool]
	SubcontractorID        db.Col[*WarehouseTable, int32]
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
	ID      int16                  `ms:"i" json:"id" cb:"id,1"`
	Name    string                 `ms:"n" json:"nm" cb:"nm,2"`
	RowCant int8                   `ms:"r" json:"rc" cb:"rc,3"`
	ColCant int8                   `ms:"c" json:"cc" cb:"cc,4"`
	Bloques []WarehouseLayoutBlock `ms:"b" json:"bl" cb:"bl,5"`
}

type WarehouseLayoutBlock struct {
	Row    int8   `json:"rw" ms:"r" cb:"rw,1"`
	Column int8   `json:"co" ms:"c" cb:"co,2"`
	Name   string `json:"nm" ms:"n" cb:"nm,3"`
}

type Site struct {
	db.TableStruct[SiteTable, Site]
	CompanyID   int32  `db:",pk" cb:"1"`
	ID          int32  `db:",pk" cb:"2"`
	Name        string `cb:"3"`
	Description string `cb:"4"`
	Address     string `cb:"5"`
	CityID      int32  `cb:"6"`
	City        string `db:"-" cb:"7"`
	Status      int8   `json:"ss" cb:"ss,8"`
	Updated     int32  `json:"upd" cb:"upd,9"`
	Version     int16  `json:"ver" cb:"ver,10"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,11"`
	UpdatedBy int32   `cb:"12"`
	Created   int32   `cb:"13"`
	CreatedBy int32   `cb:"14"`
	// IsPickupPoint lists the site as a store where online orders can be picked up. The
	// storefront reads the pickup sites from the public company file (cloud.PublishCompanyPub).
	IsPickupPoint bool `cb:"15"`
}

type SiteTable struct {
	db.TableStruct[SiteTable, Site]
	CompanyID     db.Col[*SiteTable, int32]
	ID            db.Col[*SiteTable, int32]
	Name          db.Col[*SiteTable, string]
	Description   db.Col[*SiteTable, string]
	Address       db.Col[*SiteTable, string]
	CityID        db.Col[*SiteTable, int32]
	Status        db.Col[*SiteTable, int8]
	Updated       db.Col[*SiteTable, int32]
	Version       db.Col[*SiteTable, int16]
	UpdatedBy     db.Col[*SiteTable, int32]
	Created       db.Col[*SiteTable, int32]
	CreatedBy     db.Col[*SiteTable, int32]
	IsPickupPoint db.Col[*SiteTable, bool]
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
