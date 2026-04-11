package types

import (
	"app/core"
	"app/db"
)

const (
	// ClientProviderTypeClient identifies rows that behave as clients.
	ClientProviderTypeClient int8 = 1
	// ClientProviderTypeProvider identifies rows that behave as providers.
	ClientProviderTypeProvider int8 = 2
)

const (
	// PersonTypeNatural identifies natural persons.
	PersonTypeNatural int8 = 1
	// PersonTypeCompany identifies legal companies.
	PersonTypeCompany int8 = 2
)

type ClientProvider struct {
	db.TableStruct[ClientProviderTable, ClientProvider]
	EmpresaID        int32  `json:",omitempty"`
	ID               int32  `json:",omitempty"`
	Type             int8   `json:",omitempty"`
	Name             string `json:",omitempty"`
	RegistryNumber   string `json:",omitempty"`
	NameRegistryHash int64  `json:",omitempty"`
	PersonType       int8   `json:",omitempty"`
	Email            string `json:",omitempty"`
	CountryID        int16  `json:",omitempty"`
	CityID           string `json:",omitempty"`
	Created          int32  `json:",omitempty"`
	CreatedBy        int32  `json:",omitempty"`
	Status           int8   `json:"ss,omitempty"`
	Updated          int32  `json:"upd,omitempty"`
	UpdatedBy        int32  `json:",omitempty"`
	CacheVersion     uint8  `json:"ccv,omitempty"`
}

type ClientProviderTable struct {
	db.TableStruct[ClientProviderTable, ClientProvider]
	EmpresaID        db.Col[ClientProviderTable, int32]
	ID               db.Col[ClientProviderTable, int32]
	Type             db.Col[ClientProviderTable, int8]
	Name             db.Col[ClientProviderTable, string]
	RegistryNumber   db.Col[ClientProviderTable, string]
	NameRegistryHash db.Col[ClientProviderTable, int64]
	PersonType       db.Col[ClientProviderTable, int8]
	Email            db.Col[ClientProviderTable, string]
	CountryID        db.Col[ClientProviderTable, int16]
	CityID           db.Col[ClientProviderTable, string]
	Created          db.Col[ClientProviderTable, int32]
	CreatedBy        db.Col[ClientProviderTable, int32]
	Status           db.Col[ClientProviderTable, int8]
	Updated          db.Col[ClientProviderTable, int32]
	UpdatedBy        db.Col[ClientProviderTable, int32]
}

func (e *ClientProvider) SelfParse() {
	e.NameRegistryHash = core.HashInt64(e.RegistryNumber, core.NormalizeStringT(e.Name))
}

func (t ClientProviderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "client_provider",
		Partition:        t.EmpresaID,
		SaveCacheVersion: true,
		Keys:             []db.Coln{t.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{t.RegistryNumber}},
			{Type: db.TypeLocalIndex, Keys: []db.Coln{t.NameRegistryHash}},
			// Keep GET client-provider efficient for delta sync filtered by type.
			{Type: db.TypeView, Keys: []db.Coln{t.Type.Int32(), t.Updated.DecimalSize(8)}, KeepPart: true},
			// Keep initial sync efficient by filtering active rows for each type.
			{Type: db.TypeView, Keys: []db.Coln{t.Type.Int32(), t.Status.DecimalSize(1)}, KeepPart: true},
		},
	}
}
