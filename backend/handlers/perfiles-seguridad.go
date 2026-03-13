package handlers

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
)

func GetPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	records := []s.Perfil{}

	var err error
	if updated > 0 {
		companyUpdated := fmt.Sprintf("%d_%020d", req.Usuario.EmpresaID, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.Usuario.EmpresaID).Where("company_updated").GreaterEqual(companyUpdated).Exec()
	} else {
		activeProfiles := fmt.Sprintf("%d_%d_%020d", req.Usuario.EmpresaID, 1, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.Usuario.EmpresaID).Where("company_status_updated").GreaterEqual(activeProfiles).Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener perfiles.", err)
	}

	core.Log("Perfiles obtenidos:: ", len(records))
	core.Print(records)

	return core.MakeResponse(req, &records)
}

func PostPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	body := s.Perfil{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	isExistingPerfil := body.ID > 0
	body.EmpresaID = req.Usuario.EmpresaID
	core.Print(body)

	body.Updated = core.SUnixTime()
	body.PrepareCloudSync()
	perfilesToSave := []s.Perfil{body}
	if err = db.Insert(&perfilesToSave); err != nil {
		return req.MakeErr("Error al actualizar el perfil en ScyllaDB: " + err.Error())
	}

	body = perfilesToSave[0]
	body.PrepareCloudSync()
	if err = cloud.Insert([]s.Perfil{body}); err != nil {
		return req.MakeErr("Error al actualizar el perfil en cloud: " + err.Error())
	}

	if isExistingPerfil {
		affectedUsers := []coretypes.Usuario{}
		affectedUsersQuery := db.Query(&affectedUsers)
		affectedUsersQuery.EmpresaID.Equals(req.Usuario.EmpresaID).PerfilesIDs.Contains(body.ID)
		if err = affectedUsersQuery.AllowFilter().Exec(); err != nil {
			return req.MakeErr("Error al obtener los usuarios afectados por el perfil.", err)
		}

		core.Log("PostPerfiles:: usuarios afectados", len(affectedUsers), "perfil", body.ID)

		if len(affectedUsers) > 0 {
			allAffectedPerfilesIDs := make([]int32, 0, len(affectedUsers)*2)
			for _, affectedUser := range affectedUsers {
				allAffectedPerfilesIDs = append(allAffectedPerfilesIDs, affectedUser.PerfilesIDs...)
			}

			perfilesByID, perfilesErr := getPerfilesMapByIDs(req.Usuario.EmpresaID, allAffectedPerfilesIDs)
			if perfilesErr != nil {
				return req.MakeErr("Error al obtener los perfiles de los usuarios afectados.", perfilesErr)
			}

			// Keep the just-saved profile in-memory so recomputations use the new access list immediately.
			perfilesByID[body.ID] = body

			usersWithChangedAccesos := make([]coretypes.Usuario, 0, len(affectedUsers))

			for userIndex := range affectedUsers {
				affectedUser := &affectedUsers[userIndex]
				accesosComputed, accessErr := buildAccesosComputedFromPerfiles(perfilesByID, affectedUser.PerfilesIDs)
				if accessErr != nil {
					return req.MakeErr("Error al recomputar los accesos del usuario afectado.", accessErr)
				}

				previousAccesosComputed := append([]uint16{}, affectedUser.AccesosComputed...)
				nextAccesosComputed := append([]uint16{}, accesosComputed...)
				if core.CompareSlice(previousAccesosComputed, nextAccesosComputed) {
					core.Log("PostPerfiles:: usuario sin cambios", affectedUser.ID)
					continue
				}

				// Only persist the recomputed access payload for affected users.
				affectedUser.AccesosComputed = accesosComputed
				core.Log("PostPerfiles:: usuario actualizado", affectedUser.ID, "accesosComputed", len(affectedUser.AccesosComputed))
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
