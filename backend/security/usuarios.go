package security

import (
	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/security/types"
	"encoding/json"
)

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

// addAccesoNivelToGrants folds one `accesoID*10 + nivel` entry into the merge, keeping the highest
// nivel seen. Assigning a user a second profile is expected to widen what they may do, never to
// narrow it, so every merge in this file is "most permissive wins".
// addAccesoGrantToMerge folds one stored grant record into the merge, keyed by access id.
//
// Two grants of the same access — one from each of two profiles, or one from a profile and one
// direct — merge to the widest of the two: the highest nivel, and the union of the sub-accesses.
// Nothing here can drop a sub-access for having no parent, which the flat arrays this replaced had
// to guard against: a sub-access lives inside the grant that carries its access.
func addAccesoGrantToMerge(grantsByAccesoID map[int32]*core.AccesoGrant, grantRecord coreTypes.AccesoGrantRecord) {
	accesoID := int32(grantRecord.AccesoID)
	nivel := grantRecord.Nivel

	// Normalize malformed levels down to the minimum, never up: a corrupt value must not be able
	// to widen a grant.
	if nivel < 1 || nivel > 4 {
		core.Log("addAccesoGrantToMerge:: normalizando nivel", grantRecord.Nivel, "=>", accesoID, 1)
		nivel = 1
	}

	subMask := uint16(0)
	for _, subAccesoID := range grantRecord.SubAccesos {
		if subAccesoID < 1 || int32(subAccesoID) > core.MaxSubAccesoID {
			core.Log("addAccesoGrantToMerge:: sub-acceso fuera de rango, descartado", accesoID, subAccesoID)
			continue
		}
		subMask |= uint16(1) << uint16(subAccesoID-1)
	}

	accesoGrant, alreadyMerged := grantsByAccesoID[accesoID]
	if !alreadyMerged {
		grantsByAccesoID[accesoID] = &core.AccesoGrant{AccesoID: accesoID, Nivel: nivel, SubMask: subMask}
		return
	}
	if nivel > accesoGrant.Nivel {
		accesoGrant.Nivel = nivel
	}
	accesoGrant.SubMask |= subMask
}

// buildAccesosComputedFromPerfiles merges every profile assigned to a user into the two blobs the
// user row stores.
func buildAccesosComputedFromPerfiles(perfilesByID map[int32]types.Profile, profileIDs []int32) (map[int32]*core.AccesoGrant, error) {
	grantsByAccesoID := map[int32]*core.AccesoGrant{}
	if len(profileIDs) == 0 {
		core.Log("buildAccesosComputedFromPerfiles:: user sin perfiles")
		return grantsByAccesoID, nil
	}

	for _, perfilID := range profileIDs {
		profile, exists := perfilesByID[perfilID]
		if !exists {
			core.Log("buildAccesosComputedFromPerfiles:: profile no encontrado", perfilID)
			continue
		}
		core.Log("buildAccesosComputedFromPerfiles:: profile", profile.ID, "accesos", len(profile.AccesosGrants))
		for _, grantRecord := range profile.AccesosGrants {
			addAccesoGrantToMerge(grantsByAccesoID, grantRecord)
		}
	}

	return grantsByAccesoID, nil
}

// encodeMergedAccesoGrants hands the merge to the single encoder in core, which sorts it and splits
// it across the two columns.
func encodeMergedAccesoGrants(grantsByAccesoID map[int32]*core.AccesoGrant) ([]byte, []byte, error) {
	accesoGrants := make([]core.AccesoGrant, 0, len(grantsByAccesoID))
	for _, accesoGrant := range grantsByAccesoID {
		accesoGrants = append(accesoGrants, *accesoGrant)
	}

	accesosBlob, accesosSubBlob, err := core.EncodeAccesosGrants(accesoGrants)
	if err != nil {
		return nil, nil, err
	}
	core.Log("encodeMergedAccesoGrants:: accesos", len(accesoGrants),
		"bytes", len(accesosBlob), "sub-bytes", len(accesosSubBlob))
	return accesosBlob, accesosSubBlob, nil
}

func GetUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("updated")

	records := []coreTypes.User{}
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

	usuarios := []coreTypes.User{}
	// QueryCachedIDs checks cache version and only fetches stale/missing records from ScyllaDB.
	queryError := db.QueryCachedIDs(&usuarios, cachedIDs)
	if queryError != nil {
		return req.MakeErr("Error al obtener los usuarios.", queryError)
	}

	return core.MakeResponse(req, &usuarios)
}

func PostUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	body := coreTypes.User{}
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

	// Los accesos se otorgan por perfil o directamente (AccesosGrants); basta con uno de los dos.
	if body.ID != 1 && len(body.ProfileIDs) == 0 && len(body.AccesosGrants) == 0 && !isUsuarioPropio {
		return req.MakeErr("El user debe tener al menos 1 perfil o 1 acceso directo")
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
		usuariosExistentes := []coreTypes.User{}
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
		// cualquier usuario se auto-otorgaría accesos mandando ProfileIDs o AccesosGrants en el body.
		if isUsuarioPropio {
			body.ProfileIDs = usuarioActual.ProfileIDs
			body.AccesosGrants = usuarioActual.AccesosGrants
			body.User = usuarioActual.User
			body.Status = usuarioActual.Status
		}
	}

	if len(body.Password) >= 6 {
		passwordConcat := core.Env.SECRET_PHRASE + body.Password
		body.PasswordHash = core.FnvHashString64(passwordConcat, -1, 20)
	}

	// NEVER trust the client: the catalog decides which accesses and sub-accesses exist. Same check
	// a profile gets — direct grants reach the same blobs, so they earn the same scrutiny.
	if err = ValidateAccesoGrants(body.AccesosGrants); err != nil {
		return req.MakeErr(err)
	}

	perfilesByID, err := getPerfilesMapByIDs(body.CompanyID, body.ProfileIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los perfiles del user.", err)
	}

	grantsByAccesoID, err := buildAccesosComputedFromPerfiles(perfilesByID, body.ProfileIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los accesos del profile.", err)
	}
	// AccesosGrants are the accesses granted to this user directly, on top of their profiles. They
	// merge exactly like a profile's would — same shape, same function — so the same "highest nivel
	// wins, sub-accesses union" rule applies and there is no separate dedup step.
	for _, grantRecord := range body.AccesosGrants {
		addAccesoGrantToMerge(grantsByAccesoID, grantRecord)
	}
	accesosBlob, accesosSubBlob, err := encodeMergedAccesoGrants(grantsByAccesoID)
	if err != nil {
		return req.MakeErr("Error al codificar los accesos del user.", err)
	}

	body.Password = ""
	body.AccesosComputed = accesosBlob
	body.AccesosSubComputed = accesosSubBlob
	body.Updated = now
	body.UpdatedBy = req.User.ID
	core.Log("PostUsuarios:: user", body.ID, "perfiles", body.ProfileIDs,
		"accesos bytes", len(body.AccesosComputed), "sub bytes", len(body.AccesosSubComputed))
	core.Print(body)

	usuariosToSave := []coreTypes.User{body}
	if err = db.Insert(&usuariosToSave); err != nil {
		return req.MakeErr("Error al actualizar el user (SQL): " + err.Error())
	}

	body = usuariosToSave[0]
	if err = cloud.Insert([]coreTypes.User{body}); err != nil {
		return req.MakeErr("Error al actualizar el user (Cloud ORM): " + err.Error())
	}

	// fareward tiene los accesos de este user en memoria y acaban de cambiar. Sin esto seguiría
	// autorizando con los viejos hasta que expire su TTL. No es fatal: el guardado ya está hecho y
	// el TTL es el respaldo, así que un fallo se registra y no revierte nada.
	invalidateCachedUserAccess(req, body.CompanyID, body.ID)

	return req.MakeResponse(body)
}
