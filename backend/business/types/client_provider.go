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
	CompanyID        int32  `json:",omitempty"`
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
	UpdatedVersion   int32  `json:"upv,omitempty"`
	UpdatedBy        int32  `json:",omitempty"`
}

type ClientProviderTable struct {
	db.TableStruct[ClientProviderTable, ClientProvider]
	CompanyID        db.Col[ClientProviderTable, int32]
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
	UpdatedVersion   db.Col[ClientProviderTable, int32]
}

func (e *ClientProvider) SelfParse() {
	e.NameRegistryHash = core.HashInt64(e.RegistryNumber, core.NormalizeStringT(e.Name))
}

func (t ClientProviderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                 1,
		Name:               "client_provider",
		Partition:          t.CompanyID,
		SaveUpdatedVersion: true,
		// RegistryNumber (RUC/DNI) disambiguates homonyms; Type separates clients from providers.
		GenericRecord: db.GenericRecordSchema{
			Name: t.Name, S1: t.RegistryNumber, N1: t.Type,
		},
		Keys: db.Cols(t.ID.Autoincrement(0)),
		// Declared ranges let the delta index size its packed key: Status and Type need one digit
		// each, which leaves the implicit Updated slot inside a 4-byte column.
		FixedValues: []db.FixedValues{
			{Col: t.Status, Values: []int64{0, 1}},
			{Col: t.Type, Min: 1, Max: 2},
		},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(t.RegistryNumber)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(t.NameRegistryHash)},
			// One packed view serves both halves of the sync. Keys[0] is the column Delta() infers as
			// its filter: a first sync pins Status to the active value, a delta sync fans out over both
			// so the client can evict rows that were deleted.
			{Type: db.TypeDelta, Keys: db.Cols(t.Status, t.Type)},
		},
	}
}
