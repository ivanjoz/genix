package handlers

import (
	"app/cbor"
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/types"
	"encoding/json"
	"fmt"
	"time"
)

func makeUniqueAccessIDsFromPerfiles(perfiles []types.Perfil) []int32 {
	highestLevelByAccessID := map[int32]int32{}

	// Normalize perfil access entries from accesoID*10+nivel to plain access IDs for the login payload.
	for _, perfil := range perfiles {
		for _, accesoNivelID := range perfil.Accesos {
			accessID := accesoNivelID / 10
			accessLevel := accesoNivelID % 10

			if accessLevel < 1 || accessLevel > 4 {
				accessLevel = 1
			}
			if currentLevel, exists := highestLevelByAccessID[accessID]; !exists || accessLevel > currentLevel {
				highestLevelByAccessID[accessID] = accessLevel
			}
		}
	}

	uniqueAccessIDs := make([]int32, 0, len(highestLevelByAccessID))
	for accessID := range highestLevelByAccessID {
		uniqueAccessIDs = append(uniqueAccessIDs, accessID)
	}

	return core.MakeUnique(uniqueAccessIDs)
}

func PostLogin(req *core.HandlerArgs) core.HandlerResponse {

	type Login struct {
		EmpresaID int32
		Usuario   string
		Password  string
		CipherKey string
	}

	body := Login{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Usuario) < 4 || len(body.Password) < 5 || body.EmpresaID == 0 {
		return req.MakeErr("El usuario/password enviado no posee el formato correcto.")
	}
	if len(body.CipherKey) == 0 {
		return req.MakeErr("El CipherKey es necesario.")
	}

	usuarios := []coretypes.Usuario{}
	companyUserIndex := fmt.Sprintf("%d_%s", body.EmpresaID, body.Usuario)
	err = cloud.Select(&usuarios).Where("empresa_id").Equals(body.EmpresaID).Where("company_usuario").Equals(companyUserIndex).Exec()
	if err != nil {
		return req.MakeErr("Error al consultar el usuario.", err.Error())
	}

	core.Log("usuarios encontrados:: ", len(usuarios))

	if len(usuarios) == 0 {
		return req.MakeErr("No se encontró el usuario o el password es incorrecto.")
	}

	usuario := usuarios[0]
	core.Print(usuario)

	passwordHash := core.Env.SECRET_PHRASE + body.Password
	passwordHash = core.FnvHashString64(passwordHash, -1, 20)

	core.Log(passwordHash, usuario.PasswordHash)

	if passwordHash != usuario.PasswordHash {
		return req.MakeErr("No se encontró el usuario o el password es incorrecto.")
	}

	response, err := MakeUsuarioResponse(usuario, body.CipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}

func MakeUsuarioResponse(usuario coretypes.Usuario, cipherKey string) (map[string]any, error) {

	usuarioToken := core.UsuarioToken{
		EmpresaID: usuario.EmpresaID,
		ID:        usuario.ID,
		Created:   core.SUnixTime(),
		Usuario:   usuario.Usuario,
	}

	accesosIDs := []int32{}

	if usuario.ID == 1 {
		usuario.PerfilesIDs = []int32{-1}
		systemAccessIDs, err := core.GetAllEmbeddedAccesosIDs()
		if err != nil {
			return nil, core.Err("Error al obtener la lista global de accesos.", err)
		}
		accesosIDs = systemAccessIDs
	} else if len(usuario.PerfilesIDs) > 0 {
		perfiles := make([]types.Perfil, 0, len(usuario.PerfilesIDs))
		for _, perfilID := range usuario.PerfilesIDs {
			perfil, err := cloud.GetByID(types.Perfil{
				EmpresaID: usuario.EmpresaID,
				ID:        perfilID,
			})
			if err != nil {
				return nil, core.Err("Error al obtener perfiles.", err)
			}
			if perfil == nil {
				continue
			}
			perfiles = append(perfiles, *perfil)
		}

		accesosIDs = makeUniqueAccessIDsFromPerfiles(perfiles)
	}

	// Persist a deterministic keyed fingerprint in the token so auth can recompute and validate it.
	usuarioToken.Hash = core.ComputeUsuarioTokenHash(usuarioToken)

	// Encode the auth token as CBOR to keep the encrypted payload compact and schema-driven.
	usuarioTokenCBOR, err := cbor.Marshal(usuarioToken)
	if err != nil {
		return nil, core.Err("Error al serializar el Token de usuario en CBOR.", err)
	}
	core.Log("MakeUsuarioResponse:: usuarioTokenCBOR bytes", len(usuarioTokenCBOR))
	core.Log("MakeUsuarioResponse:: token hash", usuarioToken.Hash, "empresaID", usuario.EmpresaID, "usuarioID", usuario.ID, "accesosIDs", len(accesosIDs))

	// Publish the token as raw CBOR bytes in base64 so auth can decode it without extra transforms.
	core.Log("MakeUsuarioResponse:: token bytes", len(usuarioTokenCBOR))

	// Crea la informacion del usuario encriptada
	userInfo := map[string]any{
		"Usuario":      usuario.Usuario,
		"AccesosIDs":   accesosIDs,
		"RolesIDs":     []int32{},
		"Nombres":      usuario.Nombres,
		"Apellidos":    usuario.Apellidos,
		"Email":        usuario.Email,
		"DocumentoNro": usuario.DocumentoNro,
		"Cargo":        usuario.Cargo,
	}

	userInfoJson := core.ToJsonNoErr(userInfo)
	userInfoJsonEncrypted, err := core.Encrypt([]byte(userInfoJson), cipherKey)

	if err != nil {
		return nil, core.Err("Error al encriptar la información del usuario.", err)
	}

	response := map[string]any{
		"UserID":       usuario.ID,
		"UserToken":    core.BytesToBase64(usuarioTokenCBOR, true),
		"TokenExpTime": time.Now().Unix() + (4 * 60 * 40),
		"UserInfo":     core.BytesToBase64(userInfoJsonEncrypted),
		"EmpresaID":    usuario.EmpresaID,
	}

	return response, nil
}

func ReloadLogin(req *core.HandlerArgs) core.HandlerResponse {

	cipherKey := req.GetQuery("cipher-key")

	usuario, err := cloud.GetByID(coretypes.Usuario{
		EmpresaID: req.Usuario.EmpresaID,
		ID:        req.Usuario.ID,
	})
	if err != nil {
		return req.MakeErr("Error al obtener el usuario.", err)
	}
	if usuario == nil {
		return req.MakeErr("No se encontró el usuario.")
	}
	response, err := MakeUsuarioResponse(*usuario, cipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}
