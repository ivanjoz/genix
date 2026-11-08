// Client/provider upsert. This is business logic in a `types` package on purpose: `sales`
// resolves or creates the buyer while recording a sale, and a module body may not import
// another module body (see backend/docs/MODULE_BOUNDARIES.md). Living here is what lets
// `sales` call it while importing only `app/crm/types`.

package types

import (
	"app/core"
	"app/db"
	"cmp"
	"net/mail"
	"regexp"
	"slices"
	"strings"
)

var companyRegistryNumberPattern = regexp.MustCompile(`^\d{7,12}$`)

// SaveClientProvidersIn validates and writes clients/providers in the caller's transaction, and
// returns the saved rows with their IDs. A payload row matching a stored identity (registry number,
// then name hash) takes that row's ID instead of inserting a duplicate; onlyInsert leaves those
// stored rows untouched (the sale's buyer never edits a shared record).
//
// The caller must hold the company row's lock (config.Company): the identity lookups are plain
// reads, and only that lock keeps two saves from both missing a name and inserting it twice. The
// lock cannot be taken here, because config/types imports this package. The payload is not
// mutated, so a deadlock rerun starts again from it.
func SaveClientProvidersIn(transaction *db.Transaction, payload []ClientProvider, companyID int32, userID int32, onlyInsert bool) ([]ClientProvider, error) {
	savedPayload := slices.Clone(payload)
	clientProvidersPayload := &savedPayload
	currentTimestamp := core.SUnixTime()
	clientProviderRegistryNumbers := core.SliceSet[string]{}
	clientProviderHashes := core.SliceSet[int64]{}

	for clientProviderIndex := range *clientProvidersPayload {
		clientProvider := &(*clientProvidersPayload)[clientProviderIndex]

		// Normalize text fields before validation so the backend owns the final persisted shape.
		clientProvider.Name = strings.TrimSpace(clientProvider.Name)
		clientProvider.Email = strings.TrimSpace(strings.ToLower(clientProvider.Email))
		clientProvider.RegistryNumber = strings.TrimSpace(clientProvider.RegistryNumber)
		clientProvider.CityID = strings.TrimSpace(clientProvider.CityID)

		if clientProvider.Type != ClientProviderTypeClient && clientProvider.Type != ClientProviderTypeProvider {
			return nil, core.Err("El registro en posición", clientProviderIndex, "tiene un Type inválido. Use 1 (cliente) o 2 (proveedor).")
		}

		if len(clientProvider.Name) == 0 {
			return nil, core.Err("El registro en posición", clientProviderIndex, "debe tener Name.")
		}

		if clientProvider.PersonType != PersonTypeNatural && clientProvider.PersonType != PersonTypeCompany {
			return nil, core.Err("El registro en posición", clientProviderIndex, "tiene PersonType inválido. Use 1 (persona) o 2 (company).")
		}

		if len(clientProvider.Email) > 0 && !isValidEmailAddress(clientProvider.Email) {
			return nil, core.Err("El registro en posición", clientProviderIndex, "tiene un Email inválido.")
		}

		// A row with no declared document type falls back to the shape of its registry
		// number, so the sales book always has something to print in column 11. Only what
		// the client actually sent is validated — the fallback is ours and is known good.
		if clientProvider.IdentityDocType == 0 {
			clientProvider.IdentityDocType = DeriveIdentityDocType(clientProvider.RegistryNumber)
		} else if !IsIdentityDocType(clientProvider.IdentityDocType) {
			return nil, core.Err("El registro en posición", clientProviderIndex,
				"tiene un IdentityDocType inválido. Use uno de la tabla SUNAT (id=código):",
				IdentityDocTypeNames())
		}

		if clientProvider.Type == ClientProviderTypeProvider && clientProvider.CountryID <= 0 {
			return nil, core.Err("El registro en posición", clientProviderIndex, "debe tener CountryID válido.")
		}

		if clientProvider.Type == ClientProviderTypeProvider && len(clientProvider.CityID) == 0 {
			return nil, core.Err("El registro en posición", clientProviderIndex, "debe tener CityID válido.")
		}

		// Providers must keep a numeric registry number; natural clients can omit it completely.
		if clientProvider.Type == ClientProviderTypeProvider && !companyRegistryNumberPattern.MatchString(clientProvider.RegistryNumber) {
			return nil, core.Err("El registro en posición", clientProviderIndex, "debe tener RegistryNumber numérico de 7 a 12 dígitos para proveedores.")
		}

		// Keep the deduplication hash server-owned so DB lookups and writes use the same identity rule.
		clientProvider.SelfParse()

		// Enforce tenant and audit fields from the authenticated user, never from the client payload.
		clientProvider.CompanyID = companyID
		clientProvider.Created = currentTimestamp
		clientProvider.CreatedBy = userID
		clientProvider.Status = 1
		clientProvider.Updated = currentTimestamp
		clientProvider.UpdatedBy = userID

		if clientProvider.RegistryNumber != "" && clientProviderRegistryNumbers.Include(clientProvider.RegistryNumber) {
			return nil, core.Err("El registro en posición", clientProviderIndex, "repite RegistryNumber dentro del payload.")
		}
		clientProviderRegistryNumbers.AddIf(clientProvider.RegistryNumber)

		if clientProviderHashes.Include(clientProvider.NameRegistryHash) {
			return nil, core.Err("El registro en posición", clientProviderIndex, "repite NameRegistryHash dentro del payload.")
		}
		clientProviderHashes.Add(clientProvider.NameRegistryHash)
	}

	// Resolve existing IDs before Merge so equal identities update instead of inserting duplicates.
	existingByRegistryNumber := []ClientProvider{}
	if !clientProviderRegistryNumbers.IsEmpty() {
		q := db.Query(&existingByRegistryNumber).AllowFilter()
		err := q.Select(q.RegistryNumber, q.ID).
			CompanyID.Equals(companyID).
			RegistryNumber.In(clientProviderRegistryNumbers.Values...).Exec()

		if err != nil {
			return nil, core.Err("Error al buscar clientes/proveedores por RegistryNumber.", err)
		}
	}

	existingByRegistryNumberMap := core.SliceToMapE(existingByRegistryNumber,
		func(e ClientProvider) string { return e.RegistryNumber })

	existingByHash := []ClientProvider{}
	if !clientProviderHashes.IsEmpty() {
		q := db.Query(&existingByHash).AllowFilter()
		err := q.Select(q.NameRegistryHash, q.ID).
			CompanyID.Equals(companyID).
			NameRegistryHash.In(clientProviderHashes.Values...).Exec()

		if err != nil {
			return nil, core.Err("Error al buscar clientes/proveedores por NameRegistryHash.", err)
		}
	}

	existingByHashMap := core.SliceToMapE(existingByHash,
		func(e ClientProvider) int64 { return e.NameRegistryHash })

	for clientProviderIndex := range *clientProvidersPayload {
		currentClientProvider := &(*clientProvidersPayload)[clientProviderIndex]
		if currentClientProvider.ID > 0 {
			continue
		}

		if existing, ok := existingByRegistryNumberMap[currentClientProvider.RegistryNumber]; ok {
			currentClientProvider.ID = existing.ID
			continue
		}

		if existing, ok := existingByHashMap[currentClientProvider.NameRegistryHash]; ok {
			currentClientProvider.ID = existing.ID
		}
	}

	// Point every row at the snapshot holding the identity it is about to store. Done before
	// the merge so the column lands in the same write, and outside the loop so one payload
	// costs one lookup rather than one per row.
	identities := make([]ClientProviderSnapshot, len(*clientProvidersPayload))
	for clientProviderIndex := range *clientProvidersPayload {
		identities[clientProviderIndex] = (*clientProvidersPayload)[clientProviderIndex].IdentitySnapshot()
	}
	snapshotIDs, snapshotError := ResolveIdentitySnapshotsIn(transaction, identities, companyID, userID)
	if snapshotError != nil {
		return nil, snapshotError
	}
	for clientProviderIndex := range *clientProvidersPayload {
		(*clientProvidersPayload)[clientProviderIndex].SnapshotID = snapshotIDs[clientProviderIndex]
	}

	// Rows with an ID are stored identities: read them under lock for the columns the payload
	// does not own, and refuse an ID the company does not have.
	storedClientProviders := []ClientProvider{}
	for _, clientProvider := range *clientProvidersPayload {
		if clientProvider.ID > 0 {
			storedClientProviders = append(storedClientProviders, ClientProvider{CompanyID: companyID, ID: clientProvider.ID})
		}
	}
	slices.SortFunc(storedClientProviders, func(a, b ClientProvider) int { return cmp.Compare(a.ID, b.ID) })
	if err := db.LockByKeys(transaction, &storedClientProviders); err != nil {
		return nil, core.Err("Error al obtener los clientes/proveedores.", err)
	}
	storedByID := core.SliceToMapE(storedClientProviders, func(e ClientProvider) int32 { return e.ID })

	newClientProviders := []ClientProvider{}
	newIndexes := []int{}
	editedClientProviders := []ClientProvider{}
	for clientProviderIndex := range *clientProvidersPayload {
		clientProvider := &(*clientProvidersPayload)[clientProviderIndex]
		if clientProvider.ID <= 0 {
			newClientProviders = append(newClientProviders, *clientProvider)
			newIndexes = append(newIndexes, clientProviderIndex)
			continue
		}
		stored, found := storedByID[clientProvider.ID]
		if !found {
			return nil, core.Err("El registro en posición", clientProviderIndex, "tiene un ID que no existe:", clientProvider.ID)
		}
		if onlyInsert {
			continue
		}
		clientProvider.Created = stored.Created
		clientProvider.CreatedBy = stored.CreatedBy
		editedClientProviders = append(editedClientProviders, *clientProvider)
	}

	if err := db.UpsertIn(transaction, &editedClientProviders); err != nil {
		return nil, core.Err("Error al actualizar clientes/proveedores.", err)
	}
	if err := db.InsertIn(transaction, &newClientProviders); err != nil {
		return nil, core.Err("Error al crear clientes/proveedores.", err)
	}
	for newIndex, payloadIndex := range newIndexes {
		(*clientProvidersPayload)[payloadIndex] = newClientProviders[newIndex]
	}
	return *clientProvidersPayload, nil
}

func isValidEmailAddress(emailAddress string) bool {
	// Enforce plain addresses only so persisted emails stay normalized and free from display-name formats.
	parsedAddress, parseError := mail.ParseAddress(emailAddress)
	return parseError == nil && parsedAddress.Address == emailAddress
}
