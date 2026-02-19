package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {
	listasIDs := req.GetQueryIntSlice("ids")

	if len(listasIDs) == 0 {
		return req.MakeErr("No se enviaron los ids de las listas a consultar.")
	}

	listaRegistrosMap := map[int32]*[]s.ListaCompartidaRegistro{}
	for _, listaID := range listasIDs {
		listaRegistrosMap[listaID] = &[]s.ListaCompartidaRegistro{}
	}
	eg := errgroup.Group{}

	for _, listaID := range listasIDs {
		updated := req.GetQueryInt64(fmt.Sprintf("id_%v", listaID))

		eg.Go(func() error {
			query := db.Query(listaRegistrosMap[listaID])
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				ListaID.Equals(listaID)
			if updated > 0 {
				query.Updated.GreaterThan(updated)
			} else {
				query.Status.Equals(1)
			}
			return query.Exec()
		})
	}

	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	response := map[string]*[]s.ListaCompartidaRegistro{}
	for id, registros := range listaRegistrosMap {
		response[fmt.Sprintf("id_%v", id)] = registros

		core.Log("Listas Compartidas Registros::", id, "|", len(*registros))
	}

	return core.MakeResponse(req, &response)
}

// PostListasCompartidas performs batch upsert by (ListaID + normalized name hash).
// Behavior summary:
//  1. Validates required fields and computes NombreHash using SelfParse.
//  2. Rejects duplicate names in the same payload for the same ListaID.
//  3. Loads existing rows by incoming (ListaID, hash) scope for the tenant.
//  4. If a colliding row is active (Status > 0) and is not the same ID, it rejects the request.
//  5. For incoming inserts (ID <= 0), if collision is only with deleted rows (Status = 0),
//     it reuses that historical ID so the operation becomes an update instead of a new insert.
//  6. Persists with db.Insert and returns TempID -> NewID mapping for all input rows.
func PostListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {

	// Deserialize input payload into records to upsert.
	records := []s.ListaCompartidaRegistro{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(records) == 0 {
		return req.MakeErr("No se recibieron registros para guardar.")
	}

	uniqueIncomingRecordKeyToIndex := map[string]int{}
	uniqueIncomingHashes := []int32{}
	seenIncomingHashes := map[int32]bool{}
	uniqueIncomingListaIDs := []int32{}
	seenIncomingListaIDs := map[int32]bool{}

	// Preserve client IDs as TempID for response mapping.
	newIDs := []s.NewIDToID{}

	for index := range records {
		e := &records[index]
		// Enforce minimum validation rules for business consistency.
		if len(e.Nombre) < 4 || e.ListaID == 0 {
			return req.MakeErr("Faltan propiedades de en uno de los registros.")
		}

		// Save original incoming ID before potential ID reuse/autoincrement.
		newIDs = append(newIDs, s.NewIDToID{TempID: e.ID})

		// Keep NombreHash consistent with model SelfParse logic.
		e.SelfParse()
		recordKey := fmt.Sprintf("%v_%v", e.ListaID, e.NombreHash)
		// Prevent repeated names in the same request payload for the same list.
		if previousIndex, duplicateHashExists := uniqueIncomingRecordKeyToIndex[recordKey]; duplicateHashExists {
			previousRecordName := records[previousIndex].Nombre
			return req.MakeErr(fmt.Sprintf(
				"Hay registros repetidos por nombre en el mismo envío para la lista %v: \"%s\" y \"%s\".",
				e.ListaID, previousRecordName, e.Nombre,
			))
		}

		// Keep a unique list of hash/lista keys for the DB collision query.
		uniqueIncomingRecordKeyToIndex[recordKey] = index
		if !seenIncomingHashes[e.NombreHash] {
			uniqueIncomingHashes = append(uniqueIncomingHashes, e.NombreHash)
			seenIncomingHashes[e.NombreHash] = true
		}
		if !seenIncomingListaIDs[e.ListaID] {
			uniqueIncomingListaIDs = append(uniqueIncomingListaIDs, e.ListaID)
			seenIncomingListaIDs[e.ListaID] = true
		}
	}

	// Load all potentially colliding rows for this tenant by incoming hash set.
	existingRecordsByHash := []s.ListaCompartidaRegistro{}
	existingRecordsQuery := db.Query(&existingRecordsByHash)
	existingRecordsQuery.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		NombreHash.In(uniqueIncomingHashes...).
		ListaID.In(uniqueIncomingListaIDs...).
		AllowFilter()

	if err = existingRecordsQuery.Exec(); err != nil {
		return req.MakeErr("Error al consultar registros existentes de lista compartida: " + err.Error())
	}

	existingRecordsGroupedByRecordKey := core.SliceToMapP(existingRecordsByHash,
		func(e s.ListaCompartidaRegistro) string { return fmt.Sprintf("%v_%v", e.ListaID, e.NombreHash) })

	nowTime := time.Now().Unix()

	for index := range records {
		incomingRecord := &records[index]
		// Enforce tenant and audit fields on every write.
		incomingRecord.EmpresaID = req.Usuario.EmpresaID
		incomingRecord.Updated = nowTime
		incomingRecord.UpdatedBy = req.Usuario.ID

		recordKey := fmt.Sprintf("%v_%v", incomingRecord.ListaID, incomingRecord.NombreHash)
		collidingExistingRecords := existingRecordsGroupedByRecordKey[recordKey]
		reusableDeletedRecordID := int32(0)

		for _, existingRecord := range collidingExistingRecords {
			// Same ID means user is updating the same logical row; not a name conflict.
			isSameRecord := incomingRecord.ID > 0 && incomingRecord.ID == existingRecord.ID
			if isSameRecord {
				continue
			}
			// Active collision is forbidden for both inserts and updates.
			if existingRecord.Status > 0 {
				return req.MakeErr(fmt.Sprintf(
					"El nombre \"%s\" ya existe en la lista %v.",
					incomingRecord.Nombre,
					incomingRecord.ListaID,
				))
			}
			// Keep one deleted-row candidate to convert insert into update.
			if incomingRecord.ID <= 0 && reusableDeletedRecordID == 0 {
				reusableDeletedRecordID = existingRecord.ID
			}
		}

		// Reuse a deleted row ID so client inserts become upserts by historical name.
		if incomingRecord.ID <= 0 && reusableDeletedRecordID > 0 {
			core.Log("PostListasCompartidas:: reutilizando ID por NombreHash", incomingRecord.ListaID, incomingRecord.NombreHash, reusableDeletedRecordID)
			incomingRecord.ID = reusableDeletedRecordID
		}
	}

	// Persist as batch upsert; ORM autoincrements IDs still <= 0.
	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al actualizar / insertar el registro de lista compartida: " + err.Error())
	}

	// Complete TempID -> NewID mapping with final IDs after persistence.
	for i := range records {
		newIDs[i].NewID = records[i].ID
	}

	return req.MakeResponse(newIDs)
}
