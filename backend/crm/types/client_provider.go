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
	CompanyID        int32  `cb:"1"`
	ID               int32  `cb:"2"`
	Type             int8   `cb:"3"`
	Name             string `cb:"4"`
	RegistryNumber   string `cb:"5"`
	NameRegistryHash int64  `cb:"6"`
	// IdentityDocType is SUNAT's tipo de documento de identidad, as the numeric id of an
	// IdentityDocOptions entry — see identity_doc_type.go. It cannot be inferred from
	// RegistryNumber: a carné de extranjería, a pasaporte and a PTP are all "not 8 and not
	// 11 digits", which is why the till offers a picker. 0 means nothing was declared.
	IdentityDocType int8 `cb:"7"`
	// SnapshotID is the identity this row currently presents, in client_provider_snapshot.
	// A document copies it when it is issued and never reads it again, which is what keeps
	// a filed book from moving when the name here is corrected.
	SnapshotID int32  `cb:"8"`
	PersonType int8   `cb:"9"`
	Email      string `cb:"10"`
	CountryID  int16  `cb:"11"`
	CityID     string `cb:"12"`
	Created    int32  `cb:"13"`
	CreatedBy  int32  `cb:"14"`
	Status     int8   `json:"ss" cb:"ss,15"`
	Updated    int32  `json:"upd" cb:"upd,16"`
	Version    int16  `json:"ver" cb:"ver,17"`
	UpdatedBy  int32  `cb:"18"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,19"`
}

type ClientProviderTable struct {
	db.TableStruct[ClientProviderTable, ClientProvider]
	CompanyID        db.Col[*ClientProviderTable, int32]
	ID               db.Col[*ClientProviderTable, int32]
	Type             db.Col[*ClientProviderTable, int8]
	Name             db.Col[*ClientProviderTable, string]
	RegistryNumber   db.Col[*ClientProviderTable, string]
	NameRegistryHash db.Col[*ClientProviderTable, int64]
	IdentityDocType  db.Col[*ClientProviderTable, int8]
	SnapshotID       db.Col[*ClientProviderTable, int32]
	PersonType       db.Col[*ClientProviderTable, int8]
	Email            db.Col[*ClientProviderTable, string]
	CountryID        db.Col[*ClientProviderTable, int16]
	CityID           db.Col[*ClientProviderTable, string]
	Created          db.Col[*ClientProviderTable, int32]
	CreatedBy        db.Col[*ClientProviderTable, int32]
	Status           db.Col[*ClientProviderTable, int8]
	Updated          db.Col[*ClientProviderTable, int32]
	UpdatedBy        db.Col[*ClientProviderTable, int32]
	Version          db.Col[*ClientProviderTable, int16]
}

func (e *ClientProvider) SelfParse() {
	e.NameRegistryHash = core.HashInt64(e.RegistryNumber, core.NormalizeStringT(e.Name))
}

func (t ClientProviderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:         1,
		Name:       "client_provider",
		Partition:  t.CompanyID,
		CacheByIDs: true,
		// RegistryNumber (RUC/DNI) disambiguates homonyms; Type separates clients from providers.
		GenericRecord: db.GenericRecordSchema{
			Name: t.Name, S1: t.RegistryNumber, N1: t.Type,
		},
		Keys: db.Cols(t.ID.Autoincrement(0)),
		// Clients and providers share the index: a search by name or RUC/DNI keeps the hits of its
		// Type itself (the agent's client names, ResolveRecords). Only active rows are searchable.
		TextSearch: &db.TextSearch{Columns: db.Cols(t.Name, t.RegistryNumber), Status: t.Status},
		// Declared ranges let the delta index size its packed listing view.
		FixedValues: []db.FixedValues{
			{Col: t.Status, Values: []int64{0, 1}},
			{Col: t.Type, Min: 1, Max: 2},
		},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(t.RegistryNumber)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(t.NameRegistryHash)},
			// Type is pinned (one log per type, as the route asks for one); Status, the last key,
			// is the sync filter a first sync lists.
			{Type: db.TypeDelta, Keys: db.Cols(t.Type, t.Status)},
		},
	}
}
