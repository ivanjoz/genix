package security

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	securityTypes "app/security/types"
	"encoding/json"
	"fmt"
)

func GetPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	records := []securityTypes.Profile{}

	var err error
	if updated > 0 {
		companyUpdated := fmt.Sprintf("%d_%020d", req.User.CompanyID, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.User.CompanyID).Where("company_updated").GreaterEqual(companyUpdated).Exec()
	} else {
		activeProfiles := fmt.Sprintf("%d_%d_%020d", req.User.CompanyID, 1, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.User.CompanyID).Where("company_status_updated").GreaterEqual(activeProfiles).Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener perfiles.", err)
	}

	core.Log("Perfiles obtenidos:: ", len(records))
	core.Print(records)

	return core.MakeResponse(req, &records)
}

func PostPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	body := securityTypes.Profile{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	isExistingPerfil := body.ID > 0
	body.CompanyID = req.User.CompanyID
	core.Print(body)

	body.Updated = core.SUnixTime()
	body.PrepareCloudSync()
	perfilesToSave := []securityTypes.Profile{body}
	if err = db.Insert(&perfilesToSave); err != nil {
		return req.MakeErr("Error al actualizar el profile en ScyllaDB: " + err.Error())
	}

	body = perfilesToSave[0]
	body.PrepareCloudSync()
	if err = cloud.Insert([]securityTypes.Profile{body}); err != nil {
		return req.MakeErr("Error al actualizar el profile en cloud: " + err.Error())
	}

	if isExistingPerfil {
		affectedUsers := []coretypes.User{}
		affectedUsersQuery := db.Query(&affectedUsers)
		affectedUsersQuery.CompanyID.Equals(req.User.CompanyID).PerfilesIDs.Contains(body.ID)
		if err = affectedUsersQuery.AllowFilter().Exec(); err != nil {
			return req.MakeErr("Error al obtener los usuarios afectados por el profile.", err)
		}

		core.Log("PostPerfiles:: usuarios afectados", len(affectedUsers), "profile", body.ID)

		if len(affectedUsers) > 0 {
			allAffectedPerfilesIDs := make([]int32, 0, len(affectedUsers)*2)
			for _, affectedUser := range affectedUsers {
				allAffectedPerfilesIDs = append(allAffectedPerfilesIDs, affectedUser.PerfilesIDs...)
			}

			perfilesByID, perfilesErr := getPerfilesMapByIDs(req.User.CompanyID, allAffectedPerfilesIDs)
			if perfilesErr != nil {
				return req.MakeErr("Error al obtener los perfiles de los usuarios afectados.", perfilesErr)
			}

			// Keep the just-saved profile in-memory so recomputations use the new access list immediately.
			perfilesByID[body.ID] = body

			usersWithChangedAccesos := make([]coretypes.User, 0, len(affectedUsers))

			for userIndex := range affectedUsers {
				affectedUser := &affectedUsers[userIndex]
				accesosComputed, accessErr := buildAccesosComputedFromPerfiles(perfilesByID, affectedUser.PerfilesIDs)
				if accessErr != nil {
					return req.MakeErr("Error al recomputar los accesos del user afectado.", accessErr)
				}

				previousAccesosComputed := append([]uint16{}, affectedUser.AccesosComputed...)
				nextAccesosComputed := append([]uint16{}, accesosComputed...)
				if core.CompareSlice(previousAccesosComputed, nextAccesosComputed) {
					core.Log("PostPerfiles:: user sin cambios", affectedUser.ID)
					continue
				}

				// Only persist the recomputed access payload for affected users.
				affectedUser.AccesosComputed = accesosComputed
				core.Log("PostPerfiles:: user actualizado", affectedUser.ID, "accesosComputed", len(affectedUser.AccesosComputed))
				usersWithChangedAccesos = append(usersWithChangedAccesos, *affectedUser)
			}

			if len(usersWithChangedAccesos) > 0 {
				usuarioQuery := db.Query(&usersWithChangedAccesos)
				if err = db.Update(&usersWithChangedAccesos, usuarioQuery.AccesosComputed); err != nil {
					return req.MakeErr("Error al actualizar usuarios afectados en ScyllaDB: " + err.Error())
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
