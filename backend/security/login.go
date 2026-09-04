package security

import (
	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"encoding/json"
	"time"

	"github.com/ivanjoz/colbin"
)

// buildBootstrapAdminAccesos synthesizes user 1's grants from the catalog.
//
// They are built here and never persisted, and that is precisely why resolveRouteAccess lets user 1
// bypass the daemon: its stored blobs are empty, so asking fareward would deny it. The two places
// have to keep agreeing — a bypass without this synthesis leaves the admin with no access in the
// UI, and this synthesis without the bypass gets it 403s from the gate.
func buildBootstrapAdminAccesos() ([]byte, []byte, error) {
	accessHelper := core.GetEmbeddedAccessHelper()
	allAccessIDs, err := core.GetAllEmbeddedAccesosIDs()
	if err != nil {
		return nil, nil, core.Err("No se pudo cargar el catálogo de accesos para el user administrador.", err)
	}

	accesoGrants := make([]core.AccesoGrant, 0, len(allAccessIDs))
	for _, accesoID := range allAccessIDs {
		accesoGrant := core.AccesoGrant{AccesoID: accesoID, Nivel: 4}
		// "Todos" rather than the access's declared mask: it satisfies every sub-access check on
		// that access, costs one byte instead of two, and stays correct the day the catalog gains
		// another sub-access. Only accesses that declare any get it, or an access with none would
		// land in accesos_sub_computed carrying nothing.
		if accessInfo, found := accessHelper.GetAccesoInfo(accesoID); found && accessInfo.SubAccesosMask != 0 {
			accesoGrant.SubMask = 1 << (core.SubAccesoTodosID - 1)
		}
		accesoGrants = append(accesoGrants, accesoGrant)
	}

	return core.EncodeAccesosGrants(accesoGrants)
}

func PostLogin(req *core.HandlerArgs) core.HandlerResponse {

	type Login struct {
		CompanyID int32
		User      string
		Password  string
		CipherKey string
	}

	body := Login{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.User) < 4 || len(body.Password) < 5 || body.CompanyID == 0 {
		core.Print(body)
		return req.MakeErr("El user/password enviado no posee el formato correcto.")
	}
	// The CipherKey is validated by MakeUsuarioResponse, which is the only place that knows when an
	// empty one is admissible (see the UserInfoPlain branch there).

	usuarios := []coreTypes.User{}
	if cloud.IsDataMirrorEnabled() {
		// The mirror resolves the same lookup through the index declared on the "user" column.
		err = cloud.Select(&usuarios).Where("company_id").Equals(body.CompanyID).Where("user").Equals(body.User).Exec()
	} else {
		// Self-hosted mode authenticates directly against the primary database.
		userQuery := db.Query(&usuarios)
		userQuery.CompanyID.Equals(body.CompanyID).User.Equals(body.User).Limit(1)
		err = userQuery.Exec()
	}
	if err != nil {
		return req.MakeErr("Error al consultar el user.", err.Error())
	}

	core.Log("usuarios encontrados:: ", len(usuarios))

	if len(usuarios) == 0 {
		return req.MakeErr("No se encontró el user o el password es incorrecto.")
	}

	user := usuarios[0]
	core.Print(user)

	passwordHash := core.Env.SECRET_PHRASE + body.Password
	passwordHash = core.FnvHashString64(passwordHash, -1, 20)

	core.Log(passwordHash, user.PasswordHash)

	if passwordHash != user.PasswordHash {
		return req.MakeErr("No se encontró el user o el password es incorrecto.")
	}

	response, err := MakeUsuarioResponse(user, body.CipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}

func MakeUsuarioResponse(user coreTypes.User, cipherKey string) (map[string]any, error) {

	usuarioToken := core.UsuarioToken{
		CompanyID: user.CompanyID,
		ID:        user.ID,
		Created:   core.SUnixTime(),
		User:      user.User,
	}

	// The stored blobs go out as-is: they are already sorted and split by the single encoder that
	// wrote them, so the login path re-encodes nothing.
	accesosBlob, accesosSubBlob := user.AccesosComputed, user.AccesosSubComputed
	if user.ID == 1 {
		var adminErr error
		accesosBlob, accesosSubBlob, adminErr = buildBootstrapAdminAccesos()
		if adminErr != nil {
			return nil, adminErr
		}
	}

	// Persist a deterministic keyed fingerprint in the token so auth can recompute and validate it.
	usuarioToken.Hash = core.ComputeUsuarioTokenHash(usuarioToken)

	// Encode the auth token with colbin to keep the encrypted payload compact and schema-driven.
	usuarioTokenCBOR, err := colbin.Marshal(usuarioToken)
	if err != nil {
		return nil, core.Err("Error al serializar el Token de user.", err)
	}
	core.Log("MakeUsuarioResponse:: usuarioTokenCBOR bytes", len(usuarioTokenCBOR))
	core.Log("MakeUsuarioResponse:: token hash", usuarioToken.Hash, "companyID", user.CompanyID,
		"userID", user.ID, "accesos bytes", len(accesosBlob), "sub bytes", len(accesosSubBlob))

	// Publish the token as raw CBOR bytes in base64 so auth can decode it without extra transforms.
	core.Log("MakeUsuarioResponse:: token bytes", len(usuarioTokenCBOR))

	// Crea la informacion del user encriptada
	userInfo := map[string]any{
		// El ID va dentro del UserInfo cifrado —y no solo como campo suelto de la respuesta—
		// porque es lo único que el frontend persiste del usuario de sesión: parseLogin guarda
		// el UserInfo descifrado y descarta el resto. Sin él, todo lo que necesite identificar
		// al usuario en el cliente (p.ej. el token de canal del agente) queda con userID = 0.
		"ID":             user.ID,
		"User":           user.User,
		"ProfileIDs":     user.ProfileIDs,
		"FirstName":      user.FirstName,
		"LastName":       user.LastName,
		"Email":          user.Email,
		"DocumentNumber": user.DocumentNumber,
		"JobTitle":       user.JobTitle,
	}

	userInfoJson := core.ToJsonNoErr(userInfo)

	response := map[string]any{
		"UserID":             user.ID,
		"UserToken":          core.BytesToBase64(usuarioTokenCBOR, true),
		"TokenExpTime":       time.Now().Unix() + (4 * 60 * 40),
		"AccesosComputed":    core.BytesToBase64(accesosBlob, true),
		"AccesosSubComputed": core.BytesToBase64(accesosSubBlob, true),
		"CompanyID":          user.CompanyID,
	}

	// An empty CipherKey means "this browser cannot decrypt". Only a page served from a secure
	// context (https, or localhost) gets window.crypto.subtle, and serve_tailscale hands the dev
	// app out over plain http on a tailnet IP, which is neither — so AES-GCM is simply not there.
	// Such a client asks for the plaintext by sending no key.
	//
	// Granted on IS_DEV_ARG: the argument comes from `go run . dev` and start.js is the only thing
	// that passes it, so no deployed binary can be configured into answering this.
	if cipherKey == "" {
		if !core.Env.IS_DEV_ARG {
			return nil, core.Err("El CipherKey es necesario.")
		}
		core.Log("MakeUsuarioResponse:: sin CipherKey, se envía el UserInfo en claro (arranque dev)")
		response["UserInfoPlain"] = userInfoJson
		return response, nil
	}

	userInfoJsonEncrypted, err := core.Encrypt([]byte(userInfoJson), cipherKey)
	if err != nil {
		return nil, core.Err("Error al encriptar la información del user.", err)
	}
	response["UserInfo"] = core.BytesToBase64(userInfoJsonEncrypted)

	return response, nil
}

func ReloadLogin(req *core.HandlerArgs) core.HandlerResponse {

	cipherKey := req.GetQuery("cipher-key")

	var user *coreTypes.User
	var err error
	if cloud.IsDataMirrorEnabled() {
		user, err = cloud.GetByID(coreTypes.User{
			CompanyID: req.User.CompanyID,
			ID:        req.User.ID,
		})
	} else {
		// Reload the current login from Scylla when no cloud mirror is configured.
		users := []coreTypes.User{}
		userQuery := db.Query(&users)
		userQuery.CompanyID.Equals(req.User.CompanyID).ID.Equals(req.User.ID).Limit(1)
		err = userQuery.Exec()
		if err == nil && len(users) > 0 {
			user = &users[0]
		}
	}
	if err != nil {
		return req.MakeErr("Error al obtener el user.", err)
	}
	if user == nil {
		return req.MakeErr("No se encontró el user.")
	}
	response, err := MakeUsuarioResponse(*user, cipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}
