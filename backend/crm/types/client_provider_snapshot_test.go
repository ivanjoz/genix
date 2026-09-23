package types

import "testing"

func identity(name, registryNumber string, identityDocType int8, cityID string) ClientProviderSnapshot {
	snapshot := ClientProviderSnapshot{
		Name: name, RegistryNumber: registryNumber,
		IdentityDocType: identityDocType, CityID: cityID,
	}
	snapshot.SelfParse()
	return snapshot
}

// The reason the table is content-addressed rather than versioned: a name corrected and
// then corrected back has to land on the row it already had, not mint a third one.
func TestAnIdentityThatComesBackFindsItsOriginalRow(t *testing.T) {
	jane := identity("Jane", "12345678", IdentityDocDNI, "LIM")
	janne := identity("Janne", "12345678", IdentityDocDNI, "LIM")
	janeAgain := identity("Jane", "12345678", IdentityDocDNI, "LIM")

	stored := []ClientProviderSnapshot{jane, janne}
	stored[0].ID, stored[1].ID = 1, 2

	found := findIdentitySnapshot(stored, &janeAgain)
	if found == nil || found.ID != 1 {
		t.Fatalf("reverting to Jane must reuse snapshot 1, got %v", found)
	}
	if findIdentitySnapshot(stored, &janne).ID != 2 {
		t.Error("Janne must still resolve to its own row")
	}
}

// A hash collision must cost an extra row, never a document that prints another
// company's RUC. The fields are compared after the hash narrows the candidates.
func TestACollidingHashIsNotAMatch(t *testing.T) {
	stored := []ClientProviderSnapshot{identity("ACME SAC", "20000000002", IdentityDocRUC, "LIM")}
	stored[0].ID = 7

	impostor := identity("OTRA SAC", "20999999999", IdentityDocRUC, "LIM")
	impostor.Hash = stored[0].Hash // force the collision the birthday bound allows

	if found := findIdentitySnapshot(stored, &impostor); found != nil {
		t.Fatalf("a colliding hash with different fields must not match, got %+v", found)
	}
}

// The snapshot has to read back exactly what was stored, so the hash is taken over the
// raw values. Normalizing would collapse two spellings onto one row and print a name the
// document never carried.
func TestIdentitiesAreNotNormalized(t *testing.T) {
	if identity("Jane", "12345678", IdentityDocDNI, "").Hash ==
		identity("JANE", "12345678", IdentityDocDNI, "").Hash {
		t.Error("Jane and JANE must be two identities")
	}
	// Every column takes part, or an identity could change without minting a row.
	base := identity("Jane", "12345678", IdentityDocDNI, "LIM")
	for _, other := range []ClientProviderSnapshot{
		identity("Jane", "87654321", IdentityDocDNI, "LIM"),
		identity("Jane", "12345678", IdentityDocPTP, "LIM"),
		identity("Jane", "12345678", IdentityDocDNI, "ARE"),
	} {
		if other.Hash == base.Hash {
			t.Errorf("%+v must not hash like the base identity", other)
		}
	}
}

func TestIdentitySnapshotTakesTheIdentityColumnsOnly(t *testing.T) {
	client := ClientProvider{
		CompanyID: 3, ID: 91, Name: "ACME SAC", RegistryNumber: "20000000002",
		IdentityDocType: IdentityDocRUC, CityID: "LIM",
		Email: "ventas@acme.pe", PersonType: PersonTypeCompany, SnapshotID: 12,
	}
	snapshot := client.IdentitySnapshot()

	if snapshot.ID != 0 {
		t.Error("an identity carries no snapshot id of its own until it is resolved")
	}
	if !snapshot.SameIdentity(&ClientProviderSnapshot{
		Name: "ACME SAC", RegistryNumber: "20000000002",
		IdentityDocType: IdentityDocRUC, CityID: "LIM",
	}) {
		t.Errorf("identity columns not carried over: %+v", snapshot)
	}
	if snapshot.Hash == 0 {
		t.Error("IdentitySnapshot must leave the row addressable")
	}
}
