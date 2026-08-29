package security

import (
	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"encoding/binary"
	"encoding/json"
	"slices"
	"time"

	"github.com/ivanjoz/colbin"
)

func encodeAccesosComputedBase64(accesosComputed []uint16) string {
	if len(accesosComputed) == 0 {
		return ""
	}

	// Encode packed accesses as little-endian uint16 bytes so the frontend can hydrate a Uint16Array directly.
	packedAccessBytes := make([]byte, len(accesosComputed)*2)
	for index, packedAccesoNivel := range accesosComputed {
		binary.LittleEndian.PutUint16(packedAccessBytes[index*2:], packedAccesoNivel)
	}

	return core.BytesToBase64(packedAccessBytes, true)
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
	if len(body.CipherKey) == 0 {
		return req.MakeErr("El CipherKey es necesario.")
	}

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

	sortedAccesosComputed := append([]uint16{}, user.AccesosComputed...)
	if user.ID == 1 {
		// Bootstrap admin receives all declared accesses at max level in the login payload.
		allAccessIDs, err := core.GetAllEmbeddedAccesosIDs()
		if err != nil {
			return nil, core.Err("No se pudo cargar el catálogo de accesos para el user administrador.", err)
		}

		for _, accessID := range allAccessIDs {
			sortedAccesosComputed = append(sortedAccesosComputed, uint16(accessID<<2)|3)
		}
	}
	slices.Sort(sortedAccesosComputed)
	sortedAccesosComputed = slices.Compact(sortedAccesosComputed)
	accesosComputedBase64 := encodeAccesosComputedBase64(sortedAccesosComputed)

	// Persist a deterministic keyed fingerprint in the token so auth can recompute and validate it.
	usuarioToken.Hash = core.ComputeUsuarioTokenHash(usuarioToken)

	// Encode the auth token with colbin to keep the encrypted payload compact and schema-driven.
	usuarioTokenCBOR, err := colbin.Marshal(usuarioToken)
	if err != nil {
		return nil, core.Err("Error al serializar el Token de user.", err)
	}
	core.Log("MakeUsuarioResponse:: usuarioTokenCBOR bytes", len(usuarioTokenCBOR))
	core.Log("MakeUsuarioResponse:: token hash", usuarioToken.Hash, "companyID", user.CompanyID, "userID", user.ID, "accesosComputed", len(sortedAccesosComputed))

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
	userInfoJsonEncrypted, err := core.Encrypt([]byte(userInfoJson), cipherKey)

	if err != nil {
		return nil, core.Err("Error al encriptar la información del user.", err)
	}

	response := map[string]any{
		"UserID":          user.ID,
		"UserToken":       core.BytesToBase64(usuarioTokenCBOR, true),
		"TokenExpTime":    time.Now().Unix() + (4 * 60 * 40),
		"UserInfo":        core.BytesToBase64(userInfoJsonEncrypted),
		"AccesosComputed": accesosComputedBase64,
		"CompanyID":       user.CompanyID,
	}

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
