// Backfilling the identity columns the accounting books read.
//
// Three columns were added after the data existed: ClientProvider.IdentityDocType,
// ClientProvider.SnapshotID and InvoiceDocument.ClientSnapshotID. Every row written before
// them is empty, and the Registro de Ventas reads all three — so without this the book prints
// nothing for the counterpart on every historical comprobante.
//
// It is a backfill and not a read-time fallback on purpose (CLAUDE.md §3): a mapper that
// detected the old shape and repaired it would be legacy code from the day it was written, and
// the book would keep re-deriving an identity that is supposed to be frozen.
//
// What it cannot recover: the counterpart's name *as it was on the issue date*. That value was
// never stored, so a document is pinned to the identity its client presents today. From the
// pin forward the name is frozen, which is the guarantee the snapshot exists for; before it,
// the history is simply gone.

package exec

import (
	config "app/config/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	invoicing "app/invoicing/types"
	sales "app/sales/types"
	"fmt"
	"strconv"
	"strings"
)

// Snapshots this mints are stamped -1 rather than a real user id: nobody created them, and the
// value is what tells a later reader which identities were reconstructed instead of declared.
const backfillCreatedBy = int32(-1)

type identityBackfillReport struct {
	companies            int
	clientsPending       int
	clientsWritten       int
	documentsPending     int
	documentsWritten     int
	documentsAnonymous   int
	documentsWithoutSale int
}

// BackfillIdentitySnapshots fills the identity columns on clients, providers and comprobantes.
//
// fn-backfill-identity-snapshots [apply] [companyID]
//
// It reports without writing unless `apply` is given: it stamps historical accounting rows, so
// seeing the counts first is worth one extra invocation. Re-running is safe — every step keys
// off a column still being empty, and the snapshot table is content-addressed, so a second pass
// finds nothing to do and mints nothing.
func BackfillIdentitySnapshots(args *core.ExecArgs) core.FuncResponse {
	shouldWrite, onlyCompanyID := parseIdentityBackfillArguments(args.Message)

	// Every company, whatever its status: a tenant that stopped operating still has to be
	// able to produce the books for the periods it did operate in.
	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	report := identityBackfillReport{}
	for _, company := range companies {
		if onlyCompanyID > 0 && company.ID != onlyCompanyID {
			continue
		}
		report.companies++

		clientsByID, clientsError := backfillClientIdentities(company.ID, shouldWrite, &report)
		if clientsError != nil {
			return args.MakeErr(clientsError)
		}
		if documentsError := backfillDocumentIdentities(
			company.ID, clientsByID, shouldWrite, &report); documentsError != nil {
			return args.MakeErr(documentsError)
		}
	}

	mode := "SIMULACIÓN (agregue 'apply' para escribir)"
	if shouldWrite {
		mode = "APLICADO"
	}
	message := fmt.Sprintf(
		"Backfill de identidades %v: %d empresa(s) · clientes/proveedores pendientes %d, escritos %d · "+
			"comprobantes pendientes %d, escritos %d · sin cliente identificado %d · sin venta %d",
		mode, report.companies,
		report.clientsPending, report.clientsWritten,
		report.documentsPending, report.documentsWritten,
		report.documentsAnonymous, report.documentsWithoutSale,
	)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func parseIdentityBackfillArguments(rawArguments string) (bool, int32) {
	shouldWrite, onlyCompanyID := false, int32(0)
	for _, argument := range strings.Fields(rawArguments) {
		if strings.EqualFold(argument, "apply") {
			shouldWrite = true
			continue
		}
		if parsed, parseError := strconv.ParseInt(argument, 10, 32); parseError == nil {
			onlyCompanyID = int32(parsed)
		}
	}
	return shouldWrite, onlyCompanyID
}

// backfillClientIdentities derives the missing SUNAT document type and points every row at the
// snapshot holding the identity it presents, mirroring what saveClientProviders does on save.
//
// Returns every client of the company by id — the document pass needs the SnapshotID this one
// resolved, and reading the table twice would only risk the two passes disagreeing.
func backfillClientIdentities(
	companyID int32, shouldWrite bool, report *identityBackfillReport,
) (map[int32]crm.ClientProvider, error) {

	clients := []crm.ClientProvider{}
	query := db.Query(&clients)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return nil, core.Err("Error al leer los clientes/proveedores de la empresa", companyID, queryError)
	}

	pendingIndexes := []int{}
	for index := range clients {
		client := &clients[index]
		// 0 is "nothing was declared", which is exactly the state every pre-column row is in.
		derivedDocType := client.IdentityDocType
		if derivedDocType == 0 {
			derivedDocType = crm.DeriveIdentityDocType(client.RegistryNumber)
		}
		// A row that gains a document type has to be re-resolved and not merely stamped: the
		// snapshot is addressed by a hash that covers IdentityDocType, so whatever it points at
		// today was built from the empty value and is a different identity.
		if derivedDocType == client.IdentityDocType && client.SnapshotID > 0 {
			continue
		}
		client.IdentityDocType = derivedDocType
		pendingIndexes = append(pendingIndexes, index)
	}
	report.clientsPending += len(pendingIndexes)

	if len(pendingIndexes) > 0 && shouldWrite {
		identities := make([]crm.ClientProviderSnapshot, len(pendingIndexes))
		for position, index := range pendingIndexes {
			identities[position] = clients[index].IdentitySnapshot()
		}
		snapshotIDs, snapshotError := crm.ResolveIdentitySnapshots(identities, companyID, backfillCreatedBy)
		if snapshotError != nil {
			return nil, snapshotError
		}

		clientsToUpdate := make([]crm.ClientProvider, 0, len(pendingIndexes))
		for position, index := range pendingIndexes {
			clients[index].SnapshotID = snapshotIDs[position]
			clientsToUpdate = append(clientsToUpdate, clients[index])
		}
		clientTable := db.TableOf[crm.ClientProvider]()
		// Status and Type are rewritten with the values just read: the delta view is keyed on
		// (status, type, updated_version) and the ORM refuses a partial update that bumps the
		// watermark without them, because the view row would land in the wrong bucket.
		if updateError := db.Update(&clientsToUpdate,
			clientTable.IdentityDocType, clientTable.SnapshotID,
			clientTable.Status, clientTable.Type); updateError != nil {
			return nil, core.Err("Error al actualizar los clientes/proveedores de la empresa",
				companyID, updateError)
		}
		report.clientsWritten += len(clientsToUpdate)
	}

	return core.SliceToMapE(clients, func(client crm.ClientProvider) int32 { return client.ID }), nil
}

// backfillDocumentIdentities pins every comprobante that carries no identity to the one its
// buyer presents, reached through the sale the document bills.
//
// A comprobante whose sale names no client keeps its pin at 0. That is not a gap: the document
// declared CLIENTES VARIOS when it was issued, and inventing an identity for it now would make
// the book print something the comprobante never said.
func backfillDocumentIdentities(
	companyID int32, clientsByID map[int32]crm.ClientProvider,
	shouldWrite bool, report *identityBackfillReport,
) error {

	documents := []invoicing.InvoiceDocument{}
	query := db.Query(&documents)
	query.Select().CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return core.Err("Error al leer los comprobantes de la empresa", companyID, queryError)
	}

	unpinnedIndexes := []int{}
	saleOrderIDs := core.SliceSet[int64]{}
	for index := range documents {
		if documents[index].ClientSnapshotID > 0 {
			continue
		}
		unpinnedIndexes = append(unpinnedIndexes, index)
		saleOrderIDs.Add(documents[index].SaleOrderID())
	}
	if len(unpinnedIndexes) == 0 {
		return nil
	}

	orders := []sales.SaleOrder{}
	orderQuery := db.Query(&orders)
	orderQuery.Select().CompanyID.Equals(companyID).ID.In(saleOrderIDs.Values...)
	if queryError := orderQuery.Exec(); queryError != nil {
		return core.Err("Error al leer las ventas de los comprobantes de la empresa",
			companyID, queryError)
	}
	ordersByID := core.SliceToMapE(orders, func(order sales.SaleOrder) int64 { return order.ID })

	documentsToUpdate := []invoicing.InvoiceDocument{}
	for _, index := range unpinnedIndexes {
		document := &documents[index]
		order, saleFound := ordersByID[document.SaleOrderID()]
		if !saleFound {
			report.documentsWithoutSale++
			continue
		}
		client, clientFound := clientsByID[order.ClientID]
		if order.ClientID == 0 || !clientFound || client.SnapshotID == 0 {
			report.documentsAnonymous++
			continue
		}
		document.ClientSnapshotID = client.SnapshotID
		documentsToUpdate = append(documentsToUpdate, *document)
	}
	report.documentsPending += len(documentsToUpdate)

	if len(documentsToUpdate) == 0 || !shouldWrite {
		return nil
	}
	documentTable := db.TableOf[invoicing.InvoiceDocument]()
	if updateError := db.Update(&documentsToUpdate,
		documentTable.ClientSnapshotID); updateError != nil {
		return core.Err("Error al actualizar los comprobantes de la empresa", companyID, updateError)
	}
	report.documentsWritten += len(documentsToUpdate)
	return nil
}
