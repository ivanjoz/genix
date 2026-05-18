package security

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"app/security/types"
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

func getPerfilesMapByIDs(companyID int32, profileIDs []int32) (map[int32]types.Profile, error) {
	if len(profileIDs) == 0 {
		core.Log("getPerfilesMapByIDs:: sin perfiles solicitados")
		return map[int32]types.Profile{}, nil
	}

	uniquePerfilesIDs := core.MakeUnique(profileIDs)
	perfiles := []types.Profile{}
	query := db.Query(&perfiles)
	query.CompanyID.Equals(companyID).ID.In(uniquePerfilesIDs...)

	if err := query.Exec(); err != nil {
		return nil, err
	}

	core.Log("getPerfilesMapByIDs:: perfiles encontrados", len(perfiles), "de", len(uniquePerfilesIDs))

	perfilesByID := make(map[int32]types.Profile, len(perfiles))
	for _, profile := range perfiles {
		perfilesByID[profile.ID] = profile
	}

	return perfilesByID, nil
}

func buildAccesosComputedFromPerfiles(perfilesByID map[int32]types.Profile, profileIDs []int32) ([]uint16, error) {
	if len(profileIDs) == 0 {
		core.Log("buildAccesosComputedFromPerfiles:: user sin perfiles")
		return []uint16{}, nil
	}

	highestLevelByAccesoID := map[int32]int32{}

	for _, perfilID := range profileIDs {
		profile, exists := perfilesByID[perfilID]
		if !exists {
			core.Log("buildAccesosComputedFromPerfiles:: profile no encontrado", perfilID)
			continue
		}

		core.Log("buildAccesosComputedFromPerfiles:: profile", profile.ID, "accesos", len(profile.Accesos))

		for _, accesoNivelID := range profile.Accesos {
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
	companyStatusUpdated := fmt.Sprintf("%d_%d_%020d", req.User.CompanyID, 1, updated)

	records := []coretypes.User{}
	if err := cloud.Select(&records).Where("empresa_id").Equals(req.User.CompanyID).Where("company_status_updated").GreaterEqual(companyStatusUpdated).Exec(); err != nil {
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

	usuarios := []coretypes.User{}
	// QueryCachedIDs checks cache version and only fetches stale/missing records from ScyllaDB.
	queryError := db.QueryCachedIDs(&usuarios, cachedIDs)
	if queryError != nil {
		return req.MakeErr("Error al obtener los usuarios.", queryError)
	}

	return core.MakeResponse(req, &usuarios)
}

func PostUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	body := coretypes.User{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	isUsuarioPropio := req.Route == "/user-propio"
	core.Log("route::", req.Route)

	if isUsuarioPropio {
		body.ID = req.User.ID
	}

	if body.ID != 1 && len(body.PerfilesIDs) == 0 && !isUsuarioPropio {
		return req.MakeErr("El user debe tener al menos 1 permiso")
	}
	if (len(body.User) < 5 && !isUsuarioPropio) || len(body.Nombres) < 5 {
		return req.MakeErr("El user nombre debe tener al menos 5 caracteres")
	}
	if body.ID == 0 && len(body.Password) < 6 {
		return req.MakeErr("El password debe tener al menos de 6 caracteres")
	}
	if body.ID == 1 {
		body.User = "admin"
	}
	body.CompanyID = req.User.CompanyID

	now := core.SUnixTime()
	if body.ID == 0 {
		body.Created = now
		body.CreatedBy = req.User.ID
		body.Status = 1
	} else {
		usuariosExistentes := []coretypes.User{}
		query := db.Query(&usuariosExistentes)
		query.CompanyID.Equals(req.User.CompanyID).ID.Equals(body.ID).Limit(1)
		if err = query.Exec(); err != nil {
			return req.MakeErr("Error al obtener el user a actualizar.", err)
		}
		if len(usuariosExistentes) == 0 {
			return req.MakeErr("No se encontró el user a actualizar")
		}

		usuarioActual := usuariosExistentes[0]
		body.PasswordHash = usuarioActual.PasswordHash
		body.Created = usuarioActual.Created
		body.CreatedBy = usuarioActual.CreatedBy
		if isUsuarioPropio {
			body.PerfilesIDs = usuarioActual.PerfilesIDs
			body.User = usuarioActual.User
		}
	}

	if len(body.Password) >= 6 {
		passwordConcat := core.Env.SECRET_PHRASE + body.Password
		body.PasswordHash = core.FnvHashString64(passwordConcat, -1, 20)
	}

	perfilesByID, err := getPerfilesMapByIDs(body.CompanyID, body.PerfilesIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los perfiles del user.", err)
	}

	accesosComputed, err := buildAccesosComputedFromPerfiles(perfilesByID, body.PerfilesIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los accesos del profile.", err)
	}
	for _, accesoNivelID := range body.AccesosNivelIDs {
		accesosComputed = append(accesosComputed, makeAccesoNivelPacked(accesoNivelID))
	}
	accesosComputed = core.MakeUnique(accesosComputed)

	body.Password = ""
	body.AccesosComputed = accesosComputed
	body.Updated = now
	body.UpdatedBy = req.User.ID
	core.Log("PostUsuarios:: user", body.ID, "perfiles", body.PerfilesIDs, "accesosComputed", len(body.AccesosComputed))
	body.PrepareCloudSync()
	core.Print(body)

	usuariosToSave := []coretypes.User{body}
	if err = db.Insert(&usuariosToSave); err != nil {
		return req.MakeErr("Error al actualizar el user (SQL): " + err.Error())
	}

	body = usuariosToSave[0]
	body.PrepareCloudSync()
	if err = cloud.Insert([]coretypes.User{body}); err != nil {
		return req.MakeErr("Error al actualizar el user (Cloud ORM): " + err.Error())
	}

	return req.MakeResponse(body)
}
