package security

import (
	"bytes"
	"cmp"
	"slices"

	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/security/types"
	"encoding/json"
)

func GetPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	records := []types.Profile{}

	var err error
	if !cloud.IsDataMirrorEnabled() {
		// Self-hosted reads use the primary table and the same active/updated semantics as the mirror.
		profileQuery := db.Query(&records)
		profileQuery.CompanyID.Equals(req.User.CompanyID)
		if updated > 0 {
			profileQuery.Updated.GreaterEqual(int32(updated))
		} else {
			profileQuery.Status.Equals(1)
		}
		err = profileQuery.AllowFilter().Exec()
	} else if updated > 0 {
		err = cloud.Select(&records).Where("company_id").Equals(req.User.CompanyID).
			Where("updated").GreaterEqual(updated).Exec()
	} else {
		err = cloud.Select(&records).Where("company_id").Equals(req.User.CompanyID).
			Where("status").Equals(1).Where("updated").GreaterEqual(updated).Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener perfiles.", err)
	}

	core.Log("Perfiles obtenidos:: ", len(records))
	core.Print(records)

	return req.MakeResponse(core.WithDeltaReplace(records))
}

func PostPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	body := types.Profile{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	isExistingPerfil := body.ID > 0
	body.CompanyID = req.User.CompanyID
	core.Print(body)

	// NEVER trust the client: the catalog decides which sub-accesses exist, and storing one it
	// never declared would put a bit into every affected user's blob that no UI can show and no
	// handler can name. Rejecting is right rather than dropping — a profile silently saved
	// without what the operator just ticked is worse than an error.
	if err = ValidateAccesoGrants(body.AccesosGrants); err != nil {
		return req.MakeErr(err)
	}

	body.Updated = core.SUnixTime()
	// The profile and the blobs of its users commit together. The users are locked before the
	// profiles they hold are read, so a concurrent save of another of their profiles (or of the user
	// itself) is either fully seen or waits: no user is left with blobs from a stale profile.
	var usersWithChangedAccesos []coreTypes.User
	submitted := body
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		perfilesToSave := []types.Profile{submitted}
		if err := db.UpsertIn(transaction, &perfilesToSave); err != nil {
			return core.Err("Error al actualizar el profile:", err)
		}
		body = perfilesToSave[0]
		usersWithChangedAccesos = nil
		if !isExistingPerfil {
			return nil
		}
		var recomputeErr error
		usersWithChangedAccesos, recomputeErr = recomputeProfileUsersIn(transaction, body)
		return recomputeErr
	})
	if err != nil {
		return req.MakeErr(err)
	}

	if err = cloud.Insert([]types.Profile{body}); err != nil {
		return req.MakeErr("Error al actualizar el profile en cloud: " + err.Error())
	}
	// Uno por user y no el comodín de la company: esta lista es exactamente la de los blobs que
	// cambiaron, así que tirar el caché de los demás sólo les costaría una lectura para volver a la
	// misma respuesta.
	for _, affectedUser := range usersWithChangedAccesos {
		invalidateCachedUserAccess(req, affectedUser.CompanyID, affectedUser.ID)
	}
	return req.MakeResponse(body)
}

// recomputeProfileUsersIn locks the users holding the saved profile and rewrites the access blobs
// that changed, from their profiles (the saved one as the transaction wrote it) and their direct
// grants. It returns the users it rewrote.
func recomputeProfileUsersIn(transaction *db.Transaction, savedProfile types.Profile) ([]coreTypes.User, error) {
	affectedUsers := []coreTypes.User{}
	affectedUsersQuery := db.Query(&affectedUsers)
	affectedUsersQuery.Select(affectedUsersQuery.ID).
		CompanyID.Equals(savedProfile.CompanyID).ProfileIDs.Contains(savedProfile.ID)
	if err := affectedUsersQuery.AllowFilter().Exec(); err != nil {
		return nil, core.Err("Error al obtener los usuarios afectados por el profile.", err)
	}
	core.Log("PostPerfiles:: usuarios afectados", len(affectedUsers), "profile", savedProfile.ID)
	if len(affectedUsers) == 0 {
		return nil, nil
	}
	lockedUsers := make([]coreTypes.User, len(affectedUsers))
	for index, affectedUser := range affectedUsers {
		lockedUsers[index] = coreTypes.User{CompanyID: savedProfile.CompanyID, ID: affectedUser.ID}
	}
	slices.SortFunc(lockedUsers, func(left, right coreTypes.User) int { return cmp.Compare(left.ID, right.ID) })
	if err := db.LockByKeys(transaction, &lockedUsers); err != nil {
		return nil, core.Err("Error al obtener los usuarios afectados por el profile.", err)
	}

	allAffectedProfileIDs := make([]int32, 0, len(lockedUsers)*2)
	for _, lockedUser := range lockedUsers {
		allAffectedProfileIDs = append(allAffectedProfileIDs, lockedUser.ProfileIDs...)
	}
	perfilesByID, err := getPerfilesMapByIDs(savedProfile.CompanyID, allAffectedProfileIDs)
	if err != nil {
		return nil, core.Err("Error al obtener los perfiles de los usuarios afectados.", err)
	}
	// The read sees committed profiles only: the saved one is taken as this transaction wrote it.
	perfilesByID[savedProfile.ID] = savedProfile

	usersWithChangedAccesos := make([]coreTypes.User, 0, len(lockedUsers))
	for _, lockedUser := range lockedUsers {
		accesosBlob, accesosSubBlob, err := computeUserAccesos(perfilesByID, lockedUser)
		if err != nil {
			return nil, core.Err("Error al recomputar los accesos del user afectado.", err)
		}
		// Both columns decide "unchanged": a profile edit that only adds a sub-access leaves
		// accesos_computed byte-identical, and comparing just that one would skip the write and
		// leave the user with stale sub-accesses.
		if bytes.Equal(lockedUser.AccesosComputed, accesosBlob) && bytes.Equal(lockedUser.AccesosSubComputed, accesosSubBlob) {
			continue
		}
		lockedUser.AccesosComputed, lockedUser.AccesosSubComputed = accesosBlob, accesosSubBlob
		usersWithChangedAccesos = append(usersWithChangedAccesos, lockedUser)
	}
	if len(usersWithChangedAccesos) == 0 {
		return nil, nil
	}
	// Status rides along because the users table declares a composite view on (status, updated);
	// its value is the one the locked read returned, so it is current.
	usuarioQuery := db.Query(&usersWithChangedAccesos)
	if err := db.UpdateIn(transaction, &usersWithChangedAccesos,
		usuarioQuery.AccesosComputed, usuarioQuery.AccesosSubComputed, usuarioQuery.Status); err != nil {
		return nil, core.Err("Error al actualizar usuarios afectados:", err)
	}
	return usersWithChangedAccesos, nil
}

// ValidateAccesoGrants checks stored grants against the embedded catalog. A profile and a user
// grant access with the same records, so both handlers run this — which is how a user's direct
// grants came to be validated at all.
//
// The "sub-access without its access" case that the flat `accesoID*100` arrays needed a check for
// cannot be expressed here: a sub-access lives inside the grant that carries its access. What is
// left is that the access exists, that it is granted at a real level, once, and that every
// sub-access is one the catalog says that access declares.
func ValidateAccesoGrants(grantRecords []coreTypes.AccesoGrantRecord) error {
	accessHelper := core.GetEmbeddedAccessHelper()
	seenAccesoIDs := make(map[int32]bool, len(grantRecords))

	for _, grantRecord := range grantRecords {
		accesoID := int32(grantRecord.AccesoID)

		accessInfo, accesoExists := accessHelper.GetAccesoInfo(accesoID)
		if !accesoExists {
			return core.Err("El acceso", accesoID, "no existe en el catálogo.")
		}
		// Two records for one access would merge silently, and which nivel survived would depend on
		// their order — so the client is told instead.
		if seenAccesoIDs[accesoID] {
			return core.Err("El acceso", accessInfo.Name, "aparece más de una vez en los accesos otorgados.")
		}
		seenAccesoIDs[accesoID] = true

		if grantRecord.Nivel < 1 || grantRecord.Nivel > 4 {
			return core.Err("El acceso", accessInfo.Name, "tiene un nivel inválido:", grantRecord.Nivel)
		}

		for _, subAccesoID := range grantRecord.SubAccesos {
			// "Todos" is never declared in the catalog — it is synthesized — so it is allowed on any
			// access that offers sub-accesses at all, and refused on one that offers none.
			if int32(subAccesoID) == core.SubAccesoTodosID {
				if accessInfo.SubAccesosMask == 0 {
					return core.Err("El acceso", accessInfo.Name, "no declara sub-accesos.")
				}
				continue
			}
			if subAccesoID < 1 || int32(subAccesoID) > core.MaxSubAccesoID ||
				accessInfo.SubAccesosMask&(1<<uint16(subAccesoID-1)) == 0 {
				return core.Err("El acceso", accessInfo.Name, "no declara el sub-acceso", subAccesoID, ".")
			}
		}
	}

	return nil
}
