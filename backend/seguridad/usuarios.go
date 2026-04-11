package seguridad

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"app/seguridad/types"
	"encoding/json"
	"fmt"
	"sort"
)

func makeAccesoNivelUint16(accesoID int32, nivel int32) uint16 {
	// Clamp invalid levels to the minimum allowed representation to avoid granting extra permissions.
	if nivel < 1 || nivel > 4 {
		nivel = 1
	}

	return uint16(accesoID<<2) | uint16(nivel-1)
}

func makeAccesoNivelPacked(accesoNivelID int32) uint16 {
	// Reuse the same normalization path for any acceso encoded as accesoID*10+nivel.
	return makeAccesoNivelUint16(accesoNivelID/10, accesoNivelID%10)
}

func getPerfilesMapByIDs(empresaID int32, perfilesIDs []int32) (map[int32]types.Perfil, error) {
	if len(perfilesIDs) == 0 {
		core.Log("getPerfilesMapByIDs:: sin perfiles solicitados")
		return map[int32]types.Perfil{}, nil
	}

	uniquePerfilesIDs := core.MakeUnique(perfilesIDs)
	perfiles := []types.Perfil{}
	query := db.Query(&perfiles)
	query.EmpresaID.Equals(empresaID).ID.In(uniquePerfilesIDs...)

	if err := query.Exec(); err != nil {
		return nil, err
	}

	core.Log("getPerfilesMapByIDs:: perfiles encontrados", len(perfiles), "de", len(uniquePerfilesIDs))

	perfilesByID := make(map[int32]types.Perfil, len(perfiles))
	for _, perfil := range perfiles {
		perfilesByID[perfil.ID] = perfil
	}

	return perfilesByID, nil
}

func buildAccesosComputedFromPerfiles(perfilesByID map[int32]types.Perfil, perfilesIDs []int32) ([]uint16, error) {
	if len(perfilesIDs) == 0 {
		core.Log("buildAccesosComputedFromPerfiles:: usuario sin perfiles")
		return []uint16{}, nil
	}

	highestLevelByAccesoID := map[int32]int32{}

	for _, perfilID := range perfilesIDs {
		perfil, exists := perfilesByID[perfilID]
		if !exists {
			core.Log("buildAccesosComputedFromPerfiles:: perfil no encontrado", perfilID)
			continue
		}

		core.Log("buildAccesosComputedFromPerfiles:: perfil", perfil.ID, "accesos", len(perfil.Accesos))

		for _, accesoNivelID := range perfil.Accesos {
			accesoID := accesoNivelID / 10
			nivel := accesoNivelID % 10

			// Normalize malformed levels to the minimum valid level expected by the bit-packing format.
			if nivel > 4 || nivel < 1 {
				core.Log("buildAccesosComputedFromPerfiles:: normalizando nivel", accesoNivelID, "=>", accesoID, 1)
				nivel = 1
			}

			currentLevel, alreadyExists := highestLevelByAccesoID[accesoID]
			if !alreadyExists || nivel > currentLevel {
				highestLevelByAccesoID[accesoID] = nivel
			}
		}
	}

	sortedAccesoIDs := make([]int32, 0, len(highestLevelByAccesoID))
	for accesoID := range highestLevelByAccesoID {
		sortedAccesoIDs = append(sortedAccesoIDs, accesoID)
	}
	sort.Slice(sortedAccesoIDs, func(i int, j int) bool {
		return sortedAccesoIDs[i] < sortedAccesoIDs[j]
	})

	accesosComputed := make([]uint16, 0, len(sortedAccesoIDs))
	for _, accesoID := range sortedAccesoIDs {
		accesosComputed = append(accesosComputed, makeAccesoNivelPacked(accesoID*10+highestLevelByAccesoID[accesoID]))
	}

	core.Log("buildAccesosComputedFromPerfiles:: accesos computados", len(accesosComputed))

	return accesosComputed, nil
}

func GetUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("updated")
	companyStatusUpdated := fmt.Sprintf("%d_%d_%020d", req.Usuario.EmpresaID, 1, updated)

	records := []coretypes.Usuario{}
	if err := cloud.Select(&records).Where("empresa_id").Equals(req.Usuario.EmpresaID).Where("company_status_updated").GreaterEqual(companyStatusUpdated).Exec(); err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	core.Log("Usuarios obtenidos:: ", len(records))

	return core.MakeResponse(req, &records)
}

func GetUsuariosByIDs(req *core.HandlerArgs) core.HandlerResponse {
	// Parse IDs + cache versions sent by the client to resolve only changed records.
	cachedIDs := req.ExtractCacheVersionValues()

	if len(cachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	core.Log("buscando usuarios ids::", len(cachedIDs), "|", cachedIDs)

	usuarios := []coretypes.Usuario{}
	// QueryCachedIDs checks cache version and only fetches stale/missing records from ScyllaDB.
	queryError := db.QueryCachedIDs(&usuarios, cachedIDs)
	if queryError != nil {
		return req.MakeErr("Error al obtener los usuarios.", queryError)
	}

	return core.MakeResponse(req, &usuarios)
}

func PostUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	body := coretypes.Usuario{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	isUsuarioPropio := req.Route == "/usuario-propio"
	core.Log("route::", req.Route)

	if isUsuarioPropio {
		body.ID = req.Usuario.ID
	}

	if body.ID != 1 && len(body.PerfilesIDs) == 0 && !isUsuarioPropio {
		return req.MakeErr("El usuario debe tener al menos 1 permiso")
	}
	if (len(body.Usuario) < 5 && !isUsuarioPropio) || len(body.Nombres) < 5 {
		return req.MakeErr("El usuario nombre debe tener al menos 5 caracteres")
	}
	if body.ID == 0 && len(body.Password) < 6 {
		return req.MakeErr("El password debe tener al menos de 6 caracteres")
	}
	if body.ID == 1 {
		body.Usuario = "admin"
	}
	body.EmpresaID = req.Usuario.EmpresaID

	now := core.SUnixTime()
	if body.ID == 0 {
		body.Created = now
		body.CreatedBy = req.Usuario.ID
		body.Status = 1
	} else {
		usuariosExistentes := []coretypes.Usuario{}
		query := db.Query(&usuariosExistentes)
		query.EmpresaID.Equals(req.Usuario.EmpresaID).ID.Equals(body.ID).Limit(1)
		if err = query.Exec(); err != nil {
			return req.MakeErr("Error al obtener el usuario a actualizar.", err)
		}
		if len(usuariosExistentes) == 0 {
			return req.MakeErr("No se encontró el usuario a actualizar")
		}

		usuarioActual := usuariosExistentes[0]
		body.PasswordHash = usuarioActual.PasswordHash
		body.Created = usuarioActual.Created
		body.CreatedBy = usuarioActual.CreatedBy
		if isUsuarioPropio {
			body.PerfilesIDs = usuarioActual.PerfilesIDs
			body.Usuario = usuarioActual.Usuario
		}
	}

	if len(body.Password) >= 6 {
		passwordConcat := core.Env.SECRET_PHRASE + body.Password
		body.PasswordHash = core.FnvHashString64(passwordConcat, -1, 20)
	}

	perfilesByID, err := getPerfilesMapByIDs(body.EmpresaID, body.PerfilesIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los perfiles del usuario.", err)
	}

	accesosComputed, err := buildAccesosComputedFromPerfiles(perfilesByID, body.PerfilesIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los accesos del perfil.", err)
	}
	for _, accesoNivelID := range body.AccesosNivelIDs {
		accesosComputed = append(accesosComputed, makeAccesoNivelPacked(accesoNivelID))
	}
	accesosComputed = core.MakeUnique(accesosComputed)

	body.Password = ""
	body.AccesosComputed = accesosComputed
	body.Updated = now
	body.UpdatedBy = req.Usuario.ID
	core.Log("PostUsuarios:: usuario", body.ID, "perfiles", body.PerfilesIDs, "accesosComputed", len(body.AccesosComputed))
	body.PrepareCloudSync()
	core.Print(body)

	usuariosToSave := []coretypes.Usuario{body}
	if err = db.Insert(&usuariosToSave); err != nil {
		return req.MakeErr("Error al actualizar el usuario (SQL): " + err.Error())
	}

	body = usuariosToSave[0]
	body.PrepareCloudSync()
	if err = cloud.Insert([]coretypes.Usuario{body}); err != nil {
		return req.MakeErr("Error al actualizar el usuario (Cloud ORM): " + err.Error())
	}

	return req.MakeResponse(body)
}
