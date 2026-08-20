package security

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"app/security/types"
	"encoding/json"
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

	uniqueProfileIDs := core.MakeUnique(profileIDs)
	perfiles := []types.Profile{}
	query := db.Query(&perfiles)
	query.CompanyID.Equals(companyID).ID.In(uniqueProfileIDs...)

	if err := query.Exec(); err != nil {
		return nil, err
	}

	core.Log("getPerfilesMapByIDs:: perfiles encontrados", len(perfiles), "de", len(uniqueProfileIDs))

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

	records := []coretypes.User{}
	var err error
	if cloud.IsDataMirrorEnabled() {
		err = cloud.Select(&records).Where("company_id").Equals(req.User.CompanyID).
			Where("status").Equals(1).Where("updated").GreaterEqual(updated).Exec()
	} else {
		// Scylla stores the source fields directly; filtering stays scoped to the company partition.
		userQuery := db.Query(&records)
		userQuery.CompanyID.Equals(req.User.CompanyID).Status.Equals(1).Updated.GreaterEqual(int32(updated))
		err = userQuery.AllowFilter().Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	core.Log("Usuarios obtenidos:: ", len(records))

	return core.MakeResponse(req, &records)
}

func GetUsuariosByIDs(req *core.HandlerArgs) core.HandlerResponse {
	// Parse IDs + cache versions sent by the client to resolve only changed records.
	cachedIDs := req.ExtractUpdatedVersionValues()

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

	// mainHandler ya removió el "/" inicial y el prefijo de api, así que Route llega sin barra.
	isUsuarioPropio := req.Route == "user-self"
	core.Log("route::", req.Route)

	if isUsuarioPropio {
		body.ID = req.User.ID
	}

	if body.ID != 1 && len(body.ProfileIDs) == 0 && !isUsuarioPropio {
		return req.MakeErr("El user debe tener al menos 1 permiso")
	}
	if (len(body.User) < 4 && !isUsuarioPropio) || len(body.FirstName) < 4 {
		return req.MakeErr("El usuario y el nombre deben tener al menos 4 caracteres")
	}
	if body.ID == 0 && len(body.Password) < 6 {
		return req.MakeErr("El password debe tener al menos de 6 caracteres")
	}
	// User 1 is the company administrator and its login is fixed: never editable, never chosen at
	// sign-up. Forcing it here keeps the invariant even if the client sends something else.
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
		// "user-self" no exige acceso del catálogo (selfServiceRoutes en main-handlers.go), así que
		// todo lo que determina permisos se restaura desde el registro guardado: de lo contrario
		// cualquier usuario se auto-otorgaría accesos mandando ProfileIDs o AccessLevelIDs en el body.
		if isUsuarioPropio {
			body.ProfileIDs = usuarioActual.ProfileIDs
			body.AccessLevelIDs = usuarioActual.AccessLevelIDs
			body.User = usuarioActual.User
			body.Status = usuarioActual.Status
		}
	}

	if len(body.Password) >= 6 {
		passwordConcat := core.Env.SECRET_PHRASE + body.Password
		body.PasswordHash = core.FnvHashString64(passwordConcat, -1, 20)
	}

	perfilesByID, err := getPerfilesMapByIDs(body.CompanyID, body.ProfileIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los perfiles del user.", err)
	}

	accesosComputed, err := buildAccesosComputedFromPerfiles(perfilesByID, body.ProfileIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los accesos del profile.", err)
	}
	for _, accesoNivelID := range body.AccessLevelIDs {
		accesosComputed = append(accesosComputed, makeAccesoNivelPacked(accesoNivelID))
	}
	accesosComputed = core.MakeUnique(accesosComputed)

	body.Password = ""
	body.AccesosComputed = accesosComputed
	body.Updated = now
	body.UpdatedBy = req.User.ID
	core.Log("PostUsuarios:: user", body.ID, "perfiles", body.ProfileIDs, "accesosComputed", len(body.AccesosComputed))
	core.Print(body)

	usuariosToSave := []coretypes.User{body}
	if err = db.Insert(&usuariosToSave); err != nil {
		return req.MakeErr("Error al actualizar el user (SQL): " + err.Error())
	}

	body = usuariosToSave[0]
	if err = cloud.Insert([]coretypes.User{body}); err != nil {
		return req.MakeErr("Error al actualizar el user (Cloud ORM): " + err.Error())
	}

	// server_utils tiene los accesos de este user en memoria y acaban de cambiar. Sin esto seguiría
	// autorizando con los viejos hasta que expire su TTL. No es fatal: el guardado ya está hecho y
	// el TTL es el respaldo, así que un fallo se registra y no revierte nada.
	invalidateCachedUserAccess(req, body.CompanyID, body.ID)

	return req.MakeResponse(body)
}
