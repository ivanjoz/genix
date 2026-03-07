package types

import "app/db"

const (
	// EntityTypeClient identifies entities that are clients.
	EntityTypeClient int8 = 1
	// EntityTypeProvider identifies entities that are providers.
	EntityTypeProvider int8 = 2
)

const (
	// PersonTypeNatural identifies natural persons.
	PersonTypeNatural int8 = 1
	// PersonTypeCompany identifies legal companies.
	PersonTypeCompany int8 = 2
)

type Entity struct {
	db.TableStruct[EntityTable, Entity]
	EmpresaID      int32  `json:",omitempty"`
	ID             int32  `json:",omitempty"`
	Type           int8   `json:",omitempty"`
	Name           string `json:",omitempty"`
	RegistryNumber string `json:",omitempty"`
	PersonType     int8   `json:",omitempty"`
	Email          string `json:",omitempty"`
	CountryID      int16  `json:",omitempty"`
	CityID         string `json:",omitempty"`
	Status         int8   `json:"ss,omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedBy      int32  `json:",omitempty"`
}

type EntityTable struct {
	db.TableStruct[EntityTable, Entity]
	EmpresaID      db.Col[EntityTable, int32]
	ID             db.Col[EntityTable, int32]
	Type           db.Col[EntityTable, int8]
	Name           db.Col[EntityTable, string]
	RegistryNumber db.Col[EntityTable, string]
	PersonType     db.Col[EntityTable, int8]
	Email          db.Col[EntityTable, string]
	CountryID      db.Col[EntityTable, int16]
	CityID         db.Col[EntityTable, string]
	Status         db.Col[EntityTable, int8]
	Updated        db.Col[EntityTable, int32]
	UpdatedBy      db.Col[EntityTable, int32]
}

func (entityTable EntityTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "entities",
		Partition:    entityTable.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{entityTable.ID.Autoincrement(0)},
		Views: []db.View{
			// Composite view used by GET entities with mandatory type and delta by updated.
			{Cols: []db.Coln{entityTable.Type.Int32(), entityTable.Updated.DecimalSize(8)}, KeepPart: true},
			// Status view keeps initial sync (active only) query efficient.
			{Cols: []db.Coln{entityTable.Type.Int32(), entityTable.Status.DecimalSize(1)}, KeepPart: true},
		},
	}
}
