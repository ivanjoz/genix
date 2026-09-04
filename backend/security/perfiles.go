package security

import (
	"bytes"

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
		err = cloud.Select(&records).Where("empresa_id").Equals(req.User.CompanyID).
			Where("updated").GreaterEqual(updated).Exec()
	} else {
		err = cloud.Select(&records).Where("empresa_id").Equals(req.User.CompanyID).
			Where("status").Equals(1).Where("updated").GreaterEqual(updated).Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener perfiles.", err)
	}

	core.Log("Perfiles obtenidos:: ", len(records))
	core.Print(records)

	return core.MakeResponse(req, &records)
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
	if err = validateProfileSubAccesos(body); err != nil {
		return req.MakeErr(err)
	}

	body.Updated = core.SUnixTime()
	perfilesToSave := []types.Profile{body}
	if err = db.Insert(&perfilesToSave); err != nil {
		return req.MakeErr("Error al actualizar el profile en ScyllaDB: " + err.Error())
	}

	body = perfilesToSave[0]
	if err = cloud.Insert([]types.Profile{body}); err != nil {
		return req.MakeErr("Error al actualizar el profile en cloud: " + err.Error())
	}

	if isExistingPerfil {
		affectedUsers := []coreTypes.User{}
		affectedUsersQuery := db.Query(&affectedUsers)
		affectedUsersQuery.CompanyID.Equals(req.User.CompanyID).ProfileIDs.Contains(body.ID)
		if err = affectedUsersQuery.AllowFilter().Exec(); err != nil {
			return req.MakeErr("Error al obtener los usuarios afectados por el profile.", err)
		}

		core.Log("PostPerfiles:: usuarios afectados", len(affectedUsers), "profile", body.ID)

		if len(affectedUsers) > 0 {
			allAffectedProfileIDs := make([]int32, 0, len(affectedUsers)*2)
			for _, affectedUser := range affectedUsers {
				allAffectedProfileIDs = append(allAffectedProfileIDs, affectedUser.ProfileIDs...)
			}

			perfilesByID, perfilesErr := getPerfilesMapByIDs(req.User.CompanyID, allAffectedProfileIDs)
			if perfilesErr != nil {
				return req.MakeErr("Error al obtener los perfiles de los usuarios afectados.", perfilesErr)
			}

			// Keep the just-saved profile in-memory so recomputations use the new access list immediately.
			perfilesByID[body.ID] = body

			usersWithChangedAccesos := make([]coreTypes.User, 0, len(affectedUsers))

			for userIndex := range affectedUsers {
				affectedUser := &affectedUsers[userIndex]
				grantsByAccesoID, accessErr := buildAccesosComputedFromPerfiles(perfilesByID, affectedUser.ProfileIDs)
				if accessErr != nil {
					return req.MakeErr("Error al recomputar los accesos del user afectado.", accessErr)
				}
				accesosBlob, accesosSubBlob, accessErr := encodeMergedAccesoGrants(grantsByAccesoID)
				if accessErr != nil {
					return req.MakeErr("Error al codificar los accesos del user afectado.", accessErr)
				}

				// Both columns decide "unchanged": a profile edit that only adds a sub-access
				// leaves accesos_computed byte-identical, and comparing just that one would skip
				// the write and leave the user with stale sub-accesses.
				if bytes.Equal(affectedUser.AccesosComputed, accesosBlob) &&
					bytes.Equal(affectedUser.AccesosSubComputed, accesosSubBlob) {
					core.Log("PostPerfiles:: user sin cambios", affectedUser.ID)
					continue
				}

				// Only persist the recomputed access payload for affected users.
				affectedUser.AccesosComputed = accesosBlob
				affectedUser.AccesosSubComputed = accesosSubBlob
				core.Log("PostPerfiles:: user actualizado", affectedUser.ID,
					"accesos bytes", len(accesosBlob), "sub bytes", len(accesosSubBlob))
				usersWithChangedAccesos = append(usersWithChangedAccesos, *affectedUser)
			}

			if len(usersWithChangedAccesos) > 0 {
				// Status rides along because the users table declares a composite view on
				// (status, updated) and the ORM assigns updated on every write: naming only the
				// blob columns fails with "requires the columns status, updated be updated
				// together". The value is the one the read returned.
				usuarioQuery := db.Query(&usersWithChangedAccesos)
				if err = db.Update(&usersWithChangedAccesos,
					usuarioQuery.AccesosComputed, usuarioQuery.AccesosSubComputed,
					usuarioQuery.Status); err != nil {
					return req.MakeErr("Error al actualizar usuarios afectados en ScyllaDB: " + err.Error())
				}
				// Uno por user y no el comodín de la company: esta lista es exactamente la de los
				// blobs que cambiaron, así que tirar el caché de los demás sólo les costaría una
				// lectura a ScyllaDB para volver a la misma respuesta.
				for _, affectedUser := range usersWithChangedAccesos {
					invalidateCachedUserAccess(req, affectedUser.CompanyID, affectedUser.ID)
				}
			}
			/*
				if err = cloud.Insert(affectedUsers); err != nil {
					return req.MakeErr("Error al actualizar usuarios afectados en cloud: " + err.Error())
				}
			*/
		}
	}

	return req.MakeResponse(body)
}

// validateProfileSubAccesos checks every `accesoID*100 + subID` entry against the embedded catalog.
//
// Three things have to hold, and each one fails differently if it does not: the access must exist,
// the sub-access must be one that access declares, and the profile must actually grant the parent
// access. The last is what stops a sub-access from outliving the permission it qualifies — the
// merge would drop it anyway, so accepting it here would just store a grant that silently does
// nothing.
func validateProfileSubAccesos(profile types.Profile) error {
	if len(profile.SubAccesos) == 0 {
		return nil
	}

	grantedAccesoIDs := make(map[int32]bool, len(profile.Accesos))
	for _, accesoNivelID := range profile.Accesos {
		grantedAccesoIDs[accesoNivelID/10] = true
	}

	accessHelper := core.GetEmbeddedAccessHelper()
	for _, subAccesoRef := range profile.SubAccesos {
		accesoID := subAccesoRef / 100
		subAccesoID := subAccesoRef % 100

		accessInfo, accesoExists := accessHelper.GetAccesoInfo(accesoID)
		if !accesoExists {
			return core.Err("El sub-acceso", subAccesoRef, "referencia el acceso", accesoID, "que no existe.")
		}
		if !grantedAccesoIDs[accesoID] {
			return core.Err("El profile otorga el sub-acceso", subAccesoID, "de", accessInfo.Name,
				"sin otorgar el acceso en sí.")
		}
		// "Todos" is never declared in the catalog — it is synthesized — so it is allowed on any
		// access that offers sub-accesses at all, and refused on one that offers none.
		if subAccesoID == core.SubAccesoTodosID {
			if accessInfo.SubAccesosMask == 0 {
				return core.Err("El acceso", accessInfo.Name, "no declara sub-accesos.")
			}
			continue
		}
		if subAccesoID < 1 || subAccesoID > core.MaxSubAccesoID ||
			accessInfo.SubAccesosMask&(1<<uint16(subAccesoID-1)) == 0 {
			return core.Err("El acceso", accessInfo.Name, "no declara el sub-acceso", subAccesoID, ".")
		}
	}

	return nil
}
