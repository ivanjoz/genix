package types

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
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
	scylla.TableStruct[ClientProviderTable, ClientProvider]
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
	UpdatedBy        int32  `json:",omitempty"`
	CacheVersion     uint8  `json:"ccv,omitempty"`
}

type ClientProviderTable struct {
	scylla.TableStruct[ClientProviderTable, ClientProvider]
	CompanyID        scylla.Col[ClientProviderTable, int32]
	ID               scylla.Col[ClientProviderTable, int32]
	Type             scylla.Col[ClientProviderTable, int8]
	Name             scylla.Col[ClientProviderTable, string]
	RegistryNumber   scylla.Col[ClientProviderTable, string]
	NameRegistryHash scylla.Col[ClientProviderTable, int64]
	PersonType       scylla.Col[ClientProviderTable, int8]
	Email            scylla.Col[ClientProviderTable, string]
	CountryID        scylla.Col[ClientProviderTable, int16]
	CityID           scylla.Col[ClientProviderTable, string]
	Created          scylla.Col[ClientProviderTable, int32]
	CreatedBy        scylla.Col[ClientProviderTable, int32]
	Status           scylla.Col[ClientProviderTable, int8]
	Updated          scylla.Col[ClientProviderTable, int32]
	UpdatedBy        scylla.Col[ClientProviderTable, int32]
}

func (e *ClientProvider) SelfParse() {
	e.NameRegistryHash = core.HashInt64(e.RegistryNumber, core.NormalizeStringT(e.Name))
}

func (t ClientProviderTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:             "client_provider",
		Partition:        t.CompanyID,
		SaveCacheVersion: true,
		// RegistryNumber (RUC/DNI) disambiguates homonyms; Type separates clients from providers.
		GenericRecord: scylla.GenericRecordSchema{
			Name: t.Name, S1: t.RegistryNumber, N1: t.Type,
		},
		Keys: scylla.Cols(t.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(t.RegistryNumber)},
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(t.NameRegistryHash)},
			// Keep GET client-provider efficient for delta sync filtered by type.
			{Type: scylla.TypeView, Keys: scylla.Cols(t.Type.Int32(), t.Updated.DecimalSize(8))},
			// Keep initial sync efficient by filtering active rows for each type.
			{Type: scylla.TypeView, Keys: scylla.Cols(t.Type.Int32(), t.Status.DecimalSize(1))},
		},
	}
}
