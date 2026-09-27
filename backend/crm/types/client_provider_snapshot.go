// A frozen copy of who a client or provider was when a document billed them.
//
// SUNAT's Registro de Ventas and Registro de Compras print the counterpart's name and
// identity document *as of the operation*, and a filed book may not move afterwards.
// Reading CRM live would rewrite last March's book the day someone fixes a typo in a
// client's name, so a document pins a snapshot id when it is issued and reads its
// identity from there forever.
//
// The rows are content-addressed rather than versioned per client: an identity is found
// by Hash and reused, so a name going Jane -> Janne -> Jane lands back on the row it
// already had instead of minting a third one, and a save that changed nothing writes
// nothing. The price is that a snapshot names an *identity*, not a client — two clients
// with the same name, registry number, document type and city share one row — so there is
// no way back from a snapshot to its owner. The way back is the document's own ClientID
// or ProviderID.
//
// Nothing ever updates or deletes a row, which is what makes the pin trustworthy. A row
// nobody references yet is inert, and the next identity that matches it will adopt it.

package types

import (
	"app/core"
	"app/db"
)

type ClientProviderSnapshot struct {
	db.TableStruct[ClientProviderSnapshotTable, ClientProviderSnapshot]
	CompanyID int32 `json:",omitempty"`
	ID        int32 `json:",omitempty"`
	// Hash addresses the row: every identity column below, hashed together. It is what an
	// insert looks up first, so an identity the company already holds is reused.
	//
	// 64 bits, not 32, because this is a lookup key over the whole table rather than a
	// comparison against one known row: at 50k snapshots an int32 hash collides about a
	// quarter of the time, and a collision here would print another company's RUC on a
	// filed book. Callers still compare the fields before adopting a candidate, so even a
	// collision costs an extra row and never a wrong identity.
	Hash int64 `json:",omitempty"`
	// The values are hashed raw, not normalized: the snapshot must read back exactly what
	// was stored, so "Jane" and "JANE" are two identities rather than one.
	Name            string `json:",omitempty"`
	RegistryNumber  string `json:",omitempty"`
	IdentityDocType int8   `json:",omitempty"`
	CityID          string `json:",omitempty"`

	Created   int32 `json:",omitempty"`
	CreatedBy int32 `json:",omitempty"`
	// Updated and UpdatedVersion never move after the insert — nothing updates a snapshot.
	// They are here because SaveUpdatedVersion is what QueryCachedIDs requires, and reading
	// snapshots by id through the by-IDs cache is the whole access pattern of this table.
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
}

type ClientProviderSnapshotTable struct {
	db.TableStruct[ClientProviderSnapshotTable, ClientProviderSnapshot]
	CompanyID       db.Col[*ClientProviderSnapshotTable, int32]
	ID              db.Col[*ClientProviderSnapshotTable, int32]
	Hash            db.Col[*ClientProviderSnapshotTable, int64]
	Name            db.Col[*ClientProviderSnapshotTable, string]
	RegistryNumber  db.Col[*ClientProviderSnapshotTable, string]
	IdentityDocType db.Col[*ClientProviderSnapshotTable, int8]
	CityID          db.Col[*ClientProviderSnapshotTable, string]
	Created         db.Col[*ClientProviderSnapshotTable, int32]
	CreatedBy       db.Col[*ClientProviderSnapshotTable, int32]
	Updated         db.Col[*ClientProviderSnapshotTable, int32]
	UpdatedVersion  db.Col[*ClientProviderSnapshotTable, int32]
}

func (e *ClientProviderSnapshot) SelfParse() {
	e.Hash = core.HashInt64(e.Name, e.RegistryNumber, e.IdentityDocType, e.CityID)
}

// SameIdentity is the literal comparison that guards the hash lookup.
func (e *ClientProviderSnapshot) SameIdentity(other *ClientProviderSnapshot) bool {
	return e.Name == other.Name &&
		e.RegistryNumber == other.RegistryNumber &&
		e.IdentityDocType == other.IdentityDocType &&
		e.CityID == other.CityID
}

func (t ClientProviderSnapshotTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        55,
		Name:      "client_provider_snapshot",
		Partition: t.CompanyID,
		// Documents reach a snapshot by id and nothing else, so the by-IDs cache is the
		// only read path the frontend needs. No delta index: an immutable table has no
		// changes to stream and no deletions to evict.
		SaveUpdatedVersion: true,
		Keys:               db.Cols(t.ID.Autoincrement(0)),
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(t.Hash)},
		},
	}
}

// IdentitySnapshot is the identity a client or provider currently presents, as the
// snapshot row that would hold it.
func (e *ClientProvider) IdentitySnapshot() ClientProviderSnapshot {
	snapshot := ClientProviderSnapshot{
		CompanyID:       e.CompanyID,
		Name:            e.Name,
		RegistryNumber:  e.RegistryNumber,
		IdentityDocType: e.IdentityDocType,
		CityID:          e.CityID,
	}
	snapshot.SelfParse()
	return snapshot
}

// ResolveIdentitySnapshots points every identity at the snapshot row that holds it,
// inserting the ones the company does not have yet, and returns the resolved ids in the
// order the identities were given.
//
// One indexed read per identity, rather than one IN over all of them: Hash carries a local
// index, and Scylla cannot serve an IN over a secondary index — it degrades to
// ALLOW FILTERING over the whole partition, which on a table that only ever grows is the
// wrong shape forever. Batches here are one row at the till and a handful from the CRM
// form, so the loop is cheap and stays flat as the table fills.
func ResolveIdentitySnapshots(
	identities []ClientProviderSnapshot, companyID int32, userID int32,
) ([]int32, error) {

	resolvedIDs := make([]int32, len(identities))
	snapshotsToInsert := []ClientProviderSnapshot{}
	currentTimestamp := core.SUnixTime()

	for index := range identities {
		identities[index].SelfParse()

		storedID, err := findStoredIdentity(&identities[index], companyID)
		if err != nil {
			return nil, err
		}
		if storedID > 0 {
			resolvedIDs[index] = storedID
			continue
		}
		// A payload can carry the same identity twice; the first one queues it and the
		// second picks up its id below.
		if findIdentitySnapshot(snapshotsToInsert, &identities[index]) != nil {
			continue
		}
		newSnapshot := identities[index]
		newSnapshot.CompanyID = companyID
		newSnapshot.Created = currentTimestamp
		newSnapshot.CreatedBy = userID
		newSnapshot.Updated = currentTimestamp
		snapshotsToInsert = append(snapshotsToInsert, newSnapshot)
	}

	if len(snapshotsToInsert) == 0 {
		return resolvedIDs, nil
	}
	if err := db.Insert(&snapshotsToInsert); err != nil {
		return nil, core.Err("Error al guardar las identidades de cliente/proveedor.", err)
	}

	for index := range identities {
		if resolvedIDs[index] > 0 {
			continue
		}
		minted := findIdentitySnapshot(snapshotsToInsert, &identities[index])
		if minted == nil {
			return nil, core.Err("No se pudo resolver la identidad de", identities[index].Name)
		}
		resolvedIDs[index] = minted.ID
	}
	return resolvedIDs, nil
}

// findStoredIdentity reads the snapshot already holding this identity, through the Hash
// index. The candidates it returns are compared field by field, so a hash collision costs
// an extra row rather than a document that prints the wrong RUC.
func findStoredIdentity(identity *ClientProviderSnapshot, companyID int32) (int32, error) {
	candidates := []ClientProviderSnapshot{}
	query := db.Query(&candidates)
	query.Select().CompanyID.Equals(companyID).Hash.Equals(identity.Hash)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al buscar la identidad de cliente/proveedor.", err)
	}
	if found := findIdentitySnapshot(candidates, identity); found != nil {
		return found.ID, nil
	}
	return 0, nil
}

func findIdentitySnapshot(
	snapshots []ClientProviderSnapshot, wanted *ClientProviderSnapshot,
) *ClientProviderSnapshot {

	for index := range snapshots {
		if snapshots[index].Hash != wanted.Hash {
			continue
		}
		if snapshots[index].SameIdentity(wanted) {
			return &snapshots[index]
		}
	}
	return nil
}

// ProviderSnapshotID is the identity snapshot a purchase pins for its supplier: the name and
// RUC the Registro de Compras prints (fields 12-14). No supplier pins nothing.
func ProviderSnapshotID(companyID, providerID int32) (int32, error) {
	if providerID <= 0 {
		return 0, nil
	}
	providers := []ClientProvider{}
	query := db.Query(&providers)
	query.Select(query.SnapshotID).CompanyID.Equals(companyID).ID.Equals(providerID)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al leer el proveedor.", err)
	}
	if len(providers) == 0 {
		return 0, core.Err("El proveedor", providerID, "no existe.")
	}
	return providers[0].SnapshotID, nil
}
