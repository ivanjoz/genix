package types

import "app/db"

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

type ClientProviderTable struct {
	db.TableStruct[ClientProviderTable, ClientProvider]
	EmpresaID      db.Col[ClientProviderTable, int32]
	ID             db.Col[ClientProviderTable, int32]
	Type           db.Col[ClientProviderTable, int8]
	Name           db.Col[ClientProviderTable, string]
	RegistryNumber db.Col[ClientProviderTable, string]
	PersonType     db.Col[ClientProviderTable, int8]
	Email          db.Col[ClientProviderTable, string]
	CountryID      db.Col[ClientProviderTable, int16]
	CityID         db.Col[ClientProviderTable, string]
	Status         db.Col[ClientProviderTable, int8]
	Updated        db.Col[ClientProviderTable, int32]
	UpdatedBy      db.Col[ClientProviderTable, int32]
}

func (clientProviderTable ClientProviderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "client_provider",
		Partition:    clientProviderTable.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{clientProviderTable.ID.Autoincrement(0)},
		Views: []db.View{
			// Keep GET client-provider efficient for delta sync filtered by type.
			{Keys: []db.Coln{clientProviderTable.Type.Int32(), clientProviderTable.Updated.DecimalSize(8)}, KeepPart: true},
			// Keep initial sync efficient by filtering active rows for each type.
			{Keys: []db.Coln{clientProviderTable.Type.Int32(), clientProviderTable.Status.DecimalSize(1)}, KeepPart: true},
		},
	}
}
